package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewEventListener func(opt v1alpha1.Component, info ComponentInitInfo) (EventListener, error)

type EventListener interface {
	Component
	SyncEventSource(ctx context.Context, eventSource EventSource) error
	SyncConsumer(ctx context.Context, consumers Consumers) error
}

type EventSource struct {
	Name   string
	Events []Event
}

type EventType string

type Event struct {
	Name     string
	Type     EventType
	Gitlab   *EventGitlab
	Calendar *EventCalendar
}

type EventGitlab struct {
	APIServer string
	Webhook   string
	Path      string
	Events    []string
	RepoIDs   []string
}

type EventCalendar struct {
	Schedule       string
	Interval       string
	ExclusionDates []string
	Timezone       string
}

type Consumers struct {
	Name    string
	Trigger []Trigger
}

type Trigger struct {
	EventSourceName string
	EventName       string
	EventType       string
	// Filters is the condition for consuming the event.
	Filters []Filter
	Raw     string
}

type Filter struct {
	Key      string
	Value    string
	Operator string
}
