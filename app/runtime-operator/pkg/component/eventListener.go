// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// NewEventListener returns an implementation of the event listener.
// Parameter opt indicates the user-defined component option.
// Parameter info indicates the information that can help the event listener to finish operations.
type NewEventListener func(opt v1alpha1.Component, info *ComponentInitInfo) (EventListener, error)

// EventListener manages event sources and listens for events from event sources,
// manages consumers to consume events, and triggers different event logic.
// - event sources: There are multiple ways to generate events, such as events triggered by GitLab, GitHub, Calendar, Kafka, and Redis.
// - consumers: Consumers can also trigger multiple event tasks after consuming events, such as creating Kubernetes resources, sending HTTP requests, and executing custom tasks.
// - The event listening component interface provides an event listening mechanism for the side that generates the event.
// - It also provides an event consumption mechanism for the side of the trigger task.
// - The trigger task defined in the pipeline runtime resource description serves as the basis for the consumer to execute the event task.
type EventListener interface {
	// Component will be called by the syncer to run generic operations.
	Component
	// CreateEventSource creates the event source to listen to events, that support to create multi-event source.
	// Different event sources correspond to different creation methods, such as EventSourceGitlab and EventSourceCalendar.
	// Each event source generates a unique event source ID through the event source collection unique ID.
	// Additional notes:
	// - This interface can be invoked repeatedly. If the event source description changes,
	//	 the event source information in the environment needs to be updated.
	CreateEventSource(ctx context.Context, eventSourceSet EventSourceSet) error
	// DeleteEventSource deletes the event sources by event source collection unique ID.
	DeleteEventSource(ctx context.Context, UniqueID string) error
	// CreateConsumer creates the consumer to trigger the init pipeline. Support for creating multiple consumers.
	// The consumers listen to different types of event sources that generate events.
	// Additional notes:
	// - This interface can be invoked repeatedly. If the consumer description changes,
	//   the consumer information in the environment needs to be updated.
	CreateConsumer(ctx context.Context, consumer ConsumerSet) error
	// DeleteConsumer deletes all consumers by product name and name of consumer collection.
	DeleteConsumer(ctx context.Context, productName, name string) error
}

// EventSourceSet indicates the request information of the event sources collection.
type EventSourceSet struct {
	// ResourceMetaData indicates the name and owning product of the event source collection.
	ResourceMetaData
	// UniqueID is used to distinguish between event sources collection in the same cluster.
	UniqueID string
	// Events represents the collection of event sources.
	EventSources []EventSource
}

// EventSourceType indicates the type of event source.
type EventSourceType string

const (
	// EventSourceTypeCalendar indicates the event source that the type is Calendar.
	EventSourceTypeCalendar EventSourceType = "calendar"
	// EventSourceTypeGitlab indicates the event source that the type is GitLab.
	EventSourceTypeGitlab EventSourceType = "gitlab"
)

// EventSource indicates the details of the event source.
type EventSource struct {
	// Name is the name of the event source.
	Name string
	// Gitlab indicates an event source that the type is Gitlab.
	Gitlab *EventSourceGitlab
	// Calendar indicates an event source that the type is Calendar.
	Calendar *EventSourceCalendar
}

// EventSourceGitlab indicates an event source that comes from Gitlab.
type EventSourceGitlab struct {
	// APIServer is the http or https address for GitLab.
	APIServer string
	// Events represents a collection of GitLab webhook events. eg: push_events and tag_push_events
	Events []string
	// CodeRepo is an identity of the repository by Nautes.
	CodeRepo string
	// RepoID is an identity of the repository by GitLab.
	RepoID string
}

// EventSourceCalendar indicates an event source that the type is Calendar.
type EventSourceCalendar struct {
	// Schedule refers to the scheduling rule, supporting cron expressions. eg: "0 17 * * 1-5"
	Schedule string
	// Interval refers to the time interval period between two events, such as: 1s, 30m, 2h, etc.
	Interval string
	// ExclusionDates refers to the exception dates and times of the Calendar type event source,
	// and no events will be triggered during these times.
	ExclusionDates []string
	// Timezone refers to the timezone for executing the schedule.
	Timezone string
}

// ConsumerSet represents details information of consumer collection.
type ConsumerSet struct {
	// ResourceMetaData indicates the name and owning product of the consumer collection.
	ResourceMetaData
	// Account represents the machine account that used to run the consumer.
	Account MachineAccount
	// Consumers is a collection of consumers.
	Consumers []Consumer
}

// Consumer represents the details of the consumer.
type Consumer struct {
	// UniqueID represents the unique ID of the event source collection.
	UniqueID string
	// EventSourceName is the name of the event source.
	EventSourceName string
	// EventSourceType indicates event source type.
	EventSourceType EventSourceType
	// EventTypes is the specific event types in EventSourceType.
	EventTypes []string
	// Filters indicates the collection of the condition for consuming events.
	Filters []Filter
	// Task indicates details of the task that will be dealt with after consuming the event.
	Task EventTask
}

// EventTaskType represents the event task type of the triggered.
type EventTaskType string

const (
	// EventTaskTypeRaw represents the task type is original.
	// It is unchanged that is the content of the task.
	EventTaskTypeRaw EventTaskType = "raw"
)

// EventTask represents the task that contains the event of the trigger.
type EventTask struct {
	// Type represents the event task type of the triggered.
	Type EventTaskType
	// Vars represents the data description to replace in the initial content of the event task.
	Vars []InputOverWrite
	// Raw represents the original content of the event task.
	Raw interface{}
}

// Filter is the condition for consuming the event.
type Filter struct {
	// Key indicates the key of the attribute in the event source. The value is obtained from the key.
	Key string
	// Value indicates the value to be compared. It is compared with the value corresponding to the key.
	Value string
	// Comparator is only a symbol that is used to compare.
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

type NewEventSourceRuleEngine func(info *ComponentInitInfo) EventSourceSearchEngine

// RequestDataConditions is the condition for querying the path of the target data in the event source.
type RequestDataConditions struct {
	// EventType is the specific classification of messages in the event source.
	EventType string
	// EventSourceType is the type of event that triggers the event.
	EventSourceType string
	// EventListenerType is the name of the component that processes the event.
	EventListenerType string
	// RequestVar is the name of the data that needs to be obtained from the event source.
	RequestVar string
}

const (
	EventSourceVarRef      = "ref"      // The ref of the code repo that triggered the event.
	EventSourceVarCommitID = "commitID" // The commit id of the code repo that triggered event.
)

// EventSourceSearchEngine is a search engine that can find data in event sources.
type EventSourceSearchEngine interface {
	// GetTargetPathInEventSource returns the specific path of the target data in the event source based on the incoming event type, event source type, event processor name, and the data to be requested.
	GetTargetPathInEventSource(conditions RequestDataConditions) (string, error)
}

// CodeRepoEventSourceList records the list of event sources for the code repository.
// It is used to determine whether to pass default parameters to the user pipeline.
var CodeRepoEventSourceList = sets.New[EventSourceType](EventSourceTypeGitlab)
