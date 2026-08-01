package agents

import (
	"errors"
	"sort"

	"github.com/google/uuid"
)

// ErrProfessionNotFound is returned when no agent is registered for a slug.
var ErrProfessionNotFound = errors.New("profession not found")

// Agent is a profession-bound routine generator. Each professional discipline
// gets its own agent so routines can be produced consistently and in isolation.
type Agent struct {
	ID         uuid.UUID  `json:"id"`
	Profession Profession `json:"profession"`
}

// NewAgent creates an agent for a profession.
func NewAgent(p Profession) *Agent {
	return &Agent{ID: uuid.New(), Profession: p}
}

// Generate produces a daily routine for this agent's profession.
func (a *Agent) Generate(opts GenerateOptions) (*Routine, error) {
	return generate(a.Profession, opts)
}

// Registry holds one agent per profession — the fleet that lets the caller
// generate routines for all professionals at once.
type Registry struct {
	agents map[string]*Agent
	order  []string
}

// NewRegistry builds a registry from a list of professions.
func NewRegistry(professions []Profession) *Registry {
	r := &Registry{agents: make(map[string]*Agent, len(professions))}
	for _, p := range professions {
		if _, exists := r.agents[p.Slug]; exists {
			continue
		}
		r.agents[p.Slug] = NewAgent(p)
		r.order = append(r.order, p.Slug)
	}
	return r
}

// DefaultRegistry builds a registry covering every profession in the built-in
// catalog ("all professionals").
func DefaultRegistry() *Registry {
	return NewRegistry(DefaultProfessions())
}

// Get returns the agent for a profession slug.
func (r *Registry) Get(slug string) (*Agent, error) {
	a, ok := r.agents[slug]
	if !ok {
		return nil, ErrProfessionNotFound
	}
	return a, nil
}

// List returns all registered agents in insertion order.
func (r *Registry) List() []*Agent {
	out := make([]*Agent, 0, len(r.order))
	for _, slug := range r.order {
		out = append(out, r.agents[slug])
	}
	return out
}

// Professions returns the catalog backing this registry.
func (r *Registry) Professions() []Profession {
	out := make([]Profession, 0, len(r.order))
	for _, slug := range r.order {
		out = append(out, r.agents[slug].Profession)
	}
	return out
}

// Generate produces a routine for a single profession.
func (r *Registry) Generate(slug string, opts GenerateOptions) (*Routine, error) {
	a, err := r.Get(slug)
	if err != nil {
		return nil, err
	}
	return a.Generate(opts)
}

// GenerateAll produces a routine for every registered profession. This is the
// "routines generation with all professionals" entry point: one call yields a
// routine per agent, keyed by profession slug.
func (r *Registry) GenerateAll(opts GenerateOptions) (map[string]*Routine, error) {
	out := make(map[string]*Routine, len(r.agents))
	for _, a := range r.List() {
		routine, err := a.Generate(opts)
		if err != nil {
			return nil, err
		}
		out[a.Profession.Slug] = routine
	}
	return out, nil
}

// Service wires the registry to persistence and event publishing, matching the
// service pattern used elsewhere in the codebase.
type Service struct {
	registry *Registry
	store    RoutineStore
	events   EventPublisher
}

// NewService constructs a routine-generation service.
func NewService(registry *Registry, store RoutineStore, events EventPublisher) *Service {
	return &Service{registry: registry, store: store, events: events}
}

// Registry exposes the underlying agent registry.
func (s *Service) Registry() *Registry { return s.registry }

// GenerateRoutine generates, persists, and announces a routine for one profession.
func (s *Service) GenerateRoutine(slug string, opts GenerateOptions) (*Routine, error) {
	routine, err := s.registry.Generate(slug, opts)
	if err != nil {
		return nil, err
	}
	if s.store != nil {
		if err := s.store.Save(routine); err != nil {
			return nil, err
		}
	}
	if s.events != nil {
		s.events.Publish(EventRoutineGenerated{
			RoutineID:      routine.ID,
			ProfessionSlug: routine.ProfessionSlug,
			Date:           routine.Date,
			BlockCount:     len(routine.Blocks),
		})
	}
	return routine, nil
}

// GenerateForAll generates and persists routines for every profession, returning
// them sorted by profession slug for stable output.
func (s *Service) GenerateForAll(opts GenerateOptions) ([]*Routine, error) {
	byslug, err := s.registry.GenerateAll(opts)
	if err != nil {
		return nil, err
	}
	slugs := make([]string, 0, len(byslug))
	for slug := range byslug {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	out := make([]*Routine, 0, len(slugs))
	for _, slug := range slugs {
		routine := byslug[slug]
		if s.store != nil {
			if err := s.store.Save(routine); err != nil {
				return nil, err
			}
		}
		out = append(out, routine)
	}
	if s.events != nil {
		s.events.Publish(EventBatchGenerated{Count: len(out), Date: opts.normalized().Date.Format("2006-01-02")})
	}
	return out, nil
}
