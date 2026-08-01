package agents

import (
	"time"

	"github.com/google/uuid"
)

// EventPublisher is implemented by any async message bus (Kafka, NATS,
// in-process). It matches the publisher interface used across the codebase.
type EventPublisher interface {
	Publish(event any)
}

// EventRoutineGenerated is emitted when a routine is generated for a profession.
type EventRoutineGenerated struct {
	RoutineID      uuid.UUID `json:"routine_id"`
	ProfessionSlug string    `json:"profession_slug"`
	Date           string    `json:"date"`
	BlockCount     int       `json:"block_count"`
	Timestamp      time.Time `json:"timestamp"`
}

// EventBatchGenerated is emitted after routines are generated for all professionals.
type EventBatchGenerated struct {
	Count     int       `json:"count"`
	Date      string    `json:"date"`
	Timestamp time.Time `json:"timestamp"`
}
