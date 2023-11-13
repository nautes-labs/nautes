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

package argoevent

import (
	"context"

	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"

	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type CalendarEventSourceGenerator struct {
	Components     *syncer.ComponentList
	HostEntrypoint utils.EntryPoint
	Namespace      string
	K8sClient      client.Client
	DB             database.Snapshot
	User           syncer.MachineAccount
	Space          syncer.Space
}

// CreateEventSource creates an event source resource of calendar type by event source collection unique ID.
func (cg *CalendarEventSourceGenerator) CreateEventSource(ctx context.Context, eventSource syncer.EventSourceSet) error {
	es := cg.buildBaseEventSource(eventSource.UniqueID)

	_, err := controllerutil.CreateOrUpdate(ctx, cg.K8sClient, es, func() error {
		es.Spec.Calendar = cg.createCalendarSources(eventSource)
		return nil
	})
	return err
}

// DeleteEventSource deletes an event source resource of calendar type by event source collection unique ID.
func (cg *CalendarEventSourceGenerator) DeleteEventSource(ctx context.Context, uniqueID string) error {
	es := cg.buildBaseEventSource(uniqueID)

	return cg.K8sClient.Delete(ctx, es)
}

// buildBaseEventSource returns an event source resource instance by event source collection unique ID.
func (cg *CalendarEventSourceGenerator) buildBaseEventSource(uniqueID string) *eventsourcev1alpha1.EventSource {
	return &eventsourcev1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildEventSourceName(uniqueID, syncer.EventTypeCalendar),
			Namespace: cg.Namespace,
		},
	}
}

// createCalendarSources creates a cache which records calendar event sources.
func (cg *CalendarEventSourceGenerator) createCalendarSources(eventSource syncer.EventSourceSet) map[string]eventsourcev1alpha1.CalendarEventSource {
	esMap := map[string]eventsourcev1alpha1.CalendarEventSource{}

	for _, event := range eventSource.EventSources {
		if event.Calendar == nil {
			continue
		}

		esMap[event.Name] = cg.buildCalendarEventSource(*event.Calendar)
	}

	return esMap
}

// buildCalendarEventSource returns a calendar event source instance.
func (cg *CalendarEventSourceGenerator) buildCalendarEventSource(event syncer.EventSourceCalendar) eventsourcev1alpha1.CalendarEventSource {
	return eventsourcev1alpha1.CalendarEventSource{
		Schedule:       event.Schedule,
		Interval:       event.Interval,
		ExclusionDates: event.ExclusionDates,
		Timezone:       event.Timezone,
	}
}
