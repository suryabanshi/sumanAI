package agents

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// memStore is a minimal in-memory RoutineStore for tests.
type memStore struct {
	mu       sync.Mutex
	routines map[uuid.UUID]*Routine
}

func newMemStore() *memStore {
	return &memStore{routines: make(map[uuid.UUID]*Routine)}
}

func (m *memStore) Save(r *Routine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routines[r.ID] = r
	return nil
}

func (m *memStore) GetByID(id uuid.UUID) (*Routine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.routines[id]
	if !ok {
		return nil, ErrProfessionNotFound
	}
	return r, nil
}

func (m *memStore) ListByProfession(slug string) ([]*Routine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Routine
	for _, r := range m.routines {
		if r.ProfessionSlug == slug {
			out = append(out, r)
		}
	}
	return out, nil
}

type recorder struct {
	mu     sync.Mutex
	events []any
}

func (r *recorder) Publish(e any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func TestDefaultRegistryCoversCatalog(t *testing.T) {
	reg := DefaultRegistry()
	if len(reg.List()) != len(DefaultProfessions()) {
		t.Fatalf("expected %d agents, got %d", len(DefaultProfessions()), len(reg.List()))
	}
	if _, err := reg.Get("software-engineer"); err != nil {
		t.Fatalf("expected software-engineer agent: %v", err)
	}
	if _, err := reg.Get("does-not-exist"); err != ErrProfessionNotFound {
		t.Fatalf("expected ErrProfessionNotFound, got %v", err)
	}
}

func TestGenerateProducesOrderedBlocks(t *testing.T) {
	reg := DefaultRegistry()
	routine, err := reg.Generate("physician", GenerateOptions{
		Date:            time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Intensity:       IntensityBalanced,
		IncludeWellness: true,
		IncludeBreaks:   true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(routine.Blocks) == 0 {
		t.Fatal("expected blocks")
	}
	if routine.Date != "2026-08-01" {
		t.Fatalf("unexpected date %q", routine.Date)
	}
	if routine.TotalMinutes <= 0 {
		t.Fatalf("expected positive total minutes, got %d", routine.TotalMinutes)
	}

	// Blocks must be chronologically non-overlapping and ordered.
	prev := -1
	for i, b := range routine.Blocks {
		s, err := newClock(b.Start)
		if err != nil {
			t.Fatalf("block %d bad start %q: %v", i, b.Start, err)
		}
		if s.minutes < prev {
			t.Fatalf("block %d starts before previous ends (%s)", i, b.Start)
		}
		e, err := newClock(b.End)
		if err != nil {
			t.Fatalf("block %d bad end %q: %v", i, b.End, err)
		}
		prev = e.minutes
	}
}

func TestIntensityScalesDuration(t *testing.T) {
	reg := DefaultRegistry()
	light, err := reg.Generate("software-engineer", GenerateOptions{Intensity: IntensityLight})
	if err != nil {
		t.Fatal(err)
	}
	intense, err := reg.Generate("software-engineer", GenerateOptions{Intensity: IntensityIntense})
	if err != nil {
		t.Fatal(err)
	}
	if intense.TotalMinutes <= light.TotalMinutes {
		t.Fatalf("expected intense (%d) > light (%d)", intense.TotalMinutes, light.TotalMinutes)
	}
}

func TestOvernightShiftWraps(t *testing.T) {
	reg := DefaultRegistry()
	// firefighter runs an overnight 24h shift (08:00 -> 08:00).
	routine, err := reg.Generate("firefighter", GenerateOptions{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(routine.Blocks) == 0 {
		t.Fatal("expected blocks for overnight shift")
	}
	if routine.TotalMinutes <= 0 {
		t.Fatalf("expected positive total minutes, got %d", routine.TotalMinutes)
	}
}

func TestGenerateAllProfessionals(t *testing.T) {
	reg := DefaultRegistry()
	all, err := reg.GenerateAll(GenerateOptions{})
	if err != nil {
		t.Fatalf("generate all: %v", err)
	}
	if len(all) != len(reg.List()) {
		t.Fatalf("expected a routine per agent: %d vs %d", len(all), len(reg.List()))
	}
	for slug, routine := range all {
		if routine.ProfessionSlug != slug {
			t.Fatalf("routine slug mismatch: %s vs %s", routine.ProfessionSlug, slug)
		}
		if len(routine.Blocks) == 0 {
			t.Fatalf("profession %s produced no blocks", slug)
		}
	}
}

func TestServiceGenerateForAllPersistsAndEmits(t *testing.T) {
	store := newMemStore()
	rec := &recorder{}
	svc := NewService(DefaultRegistry(), store, rec)

	routines, err := svc.GenerateForAll(GenerateOptions{IncludeBreaks: true})
	if err != nil {
		t.Fatalf("generate for all: %v", err)
	}
	if len(routines) != len(DefaultProfessions()) {
		t.Fatalf("expected %d routines, got %d", len(DefaultProfessions()), len(routines))
	}
	// Results are sorted by slug for stable output.
	for i := 1; i < len(routines); i++ {
		if routines[i-1].ProfessionSlug > routines[i].ProfessionSlug {
			t.Fatalf("routines not sorted by slug at %d", i)
		}
	}
	if len(store.routines) != len(routines) {
		t.Fatalf("expected all routines persisted: %d vs %d", len(store.routines), len(routines))
	}
	if len(rec.events) == 0 {
		t.Fatal("expected a batch event")
	}
}

func TestServiceGenerateSingleEmitsEvent(t *testing.T) {
	rec := &recorder{}
	svc := NewService(DefaultRegistry(), newMemStore(), rec)
	if _, err := svc.GenerateRoutine("teacher", GenerateOptions{}); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(rec.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(rec.events))
	}
	if _, ok := rec.events[0].(EventRoutineGenerated); !ok {
		t.Fatalf("expected EventRoutineGenerated, got %T", rec.events[0])
	}
}
