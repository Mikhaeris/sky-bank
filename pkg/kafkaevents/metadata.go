package kafkaevents

import (
	"time"
	"uuid"
)

type Metadata struct {
	EventID     uuid.UUID `json:"event_id"`
	EventType   string    `json:"event_type"`
	AggregateID uuid.UUID `json:"aggregate_id"`
	OccurredAt  time.Time `json:"occurred_at"`
	Source      string    `json:"source,omitempty"`
}

type Event[T any] struct {
	Metadata Metadata `json:"metadata"`
	Payload  T        `json:"payload"`
}

type BaseEvent interface {
	GetMetadata() Metadata
}

func (e Event[T]) GetMetadata() Metadata {
	return e.Metadata
}
