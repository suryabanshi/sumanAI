package agents

import (
	"time"

	"github.com/google/uuid"
)

// BlockCategory classifies a time block so routines can be filtered, colored, or
// balanced (e.g. capping meetings, ensuring wellness).
type BlockCategory string

const (
	BlockPersonal  BlockCategory = "personal"
	BlockDeepWork  BlockCategory = "deep_work"
	BlockMeeting   BlockCategory = "meeting"
	BlockAdmin     BlockCategory = "admin"
	BlockBreak     BlockCategory = "break"
	BlockMeal      BlockCategory = "meal"
	BlockWellness  BlockCategory = "wellness"
	BlockLearning  BlockCategory = "learning"
	BlockClinical  BlockCategory = "clinical"
	BlockFieldwork BlockCategory = "fieldwork"
	BlockCreative  BlockCategory = "creative"
)

// Priority ranks a block's importance for the day.
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// TimeBlock is a single scheduled entry in a routine.
type TimeBlock struct {
	Start    string        `json:"start"` // "HH:MM", 24-hour
	End      string        `json:"end"`   // "HH:MM", 24-hour
	Title    string        `json:"title"`
	Category BlockCategory `json:"category"`
	Priority Priority      `json:"priority"`
	Notes    string        `json:"notes,omitempty"`
}

// Routine is a generated daily schedule for a profession.
type Routine struct {
	ID             uuid.UUID   `json:"id"`
	ProfessionSlug string      `json:"profession_slug"`
	ProfessionName string      `json:"profession_name"`
	Title          string      `json:"title"`
	Date           string      `json:"date"` // "YYYY-MM-DD"
	Intensity      Intensity   `json:"intensity"`
	Blocks         []TimeBlock `json:"blocks"`
	TotalMinutes   int         `json:"total_minutes"`
	GeneratedAt    time.Time   `json:"generated_at"`
}

// Intensity controls how demanding the generated routine is: it scales the
// length of work activities and how much recovery time is inserted.
type Intensity string

const (
	IntensityLight    Intensity = "light"
	IntensityBalanced Intensity = "balanced"
	IntensityIntense  Intensity = "intense"
)

func (i Intensity) scale() float64 {
	switch i {
	case IntensityLight:
		return 0.75
	case IntensityIntense:
		return 1.25
	default:
		return 1.0
	}
}

// GenerateOptions tunes how an agent builds a routine.
type GenerateOptions struct {
	Date            time.Time `json:"date"`
	Intensity       Intensity `json:"intensity"`
	IncludeWellness bool      `json:"include_wellness"`
	IncludeBreaks   bool      `json:"include_breaks"`
	FocusAreas      []string  `json:"focus_areas"`
}

func (o GenerateOptions) normalized() GenerateOptions {
	if o.Date.IsZero() {
		o.Date = time.Now()
	}
	if o.Intensity == "" {
		o.Intensity = IntensityBalanced
	}
	return o
}

// RoutineStore persists generated routines. It mirrors the store pattern used
// across the codebase; an in-memory implementation ships in the tests.
type RoutineStore interface {
	Save(routine *Routine) error
	GetByID(id uuid.UUID) (*Routine, error)
	ListByProfession(slug string) ([]*Routine, error)
}
