package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewEventListener func(opt v1alpha1.Component, info *ComponentInitInfo) (EventListener, error)

type EventListener interface {
	Component
	CreateEventSource(ctx context.Context, eventSource EventSource) error
	DeleteEventSource(ctx context.Context, UniqueID string) error
	CreateConsumer(ctx context.Context, consumer Consumers) error
	DeleteConsumer(ctx context.Context, productName, name string) error
}

type EventSource struct {
	Resource
	// UniqueID is used to distinguish between eventsources in same cluster.
	UniqueID string
	Events   []Event
}

type EventType string

const (
	EventTypeCalendar EventType = "calendar"
	EventTypeGitlab   EventType = "gitlab"
)

type Event struct {
	Name     string
	Gitlab   *EventGitlab
	Calendar *EventCalendar
}

type EventGitlab struct {
	APIServer string
	Events    []string
	CodeRepo  string
	RepoID    string
}

type EventCalendar struct {
	Schedule       string
	Interval       string
	ExclusionDates []string
	Timezone       string
}

type Consumers struct {
	Resource
	User      User
	Consumers []Consumer
}

type Consumer struct {
	// UniqueID is the unique ID of eventsource.
	UniqueID  string
	EventName string
	EventType EventType
	// Filters is the condition for consuming the event.
	Filters []Filter
	Task    EventTask
}

type EventTaskType string

const (
	EventTaskTypeRaw EventTaskType = "raw"
)

type EventTask struct {
	Type EventTaskType
	Vars []VariableTransmission
	Raw  string
}

type VariableTransmission struct {
	Source      string
	Value       string
	Destination string
}

type Filter struct {
	Key        string
	Value      string
	Comparator Comparator
}

type Comparator string

const (
	GreaterThanOrEqualTo Comparator = ">=" // Greater than or equal to value provided in consumer
	GreaterThan          Comparator = ">"  // Greater than value provided in consumer
	EqualTo              Comparator = "="  // Equal to value provided in consumer
	NotEqualTo           Comparator = "!=" // Not equal to value provided in consumer
	LessThan             Comparator = "<"  // Less than value provided in consumer
	LessThanOrEqualTo    Comparator = "<=" // Less than or equal to value provided in consumer
	Match                Comparator = "match"
)
