package store

import (
	"fmt"
	"sort"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

type storedEvent struct {
	domain.EventLog
	TenantID string
	ServerID string
}

func seedEventsFromServices(services []domain.Service) []storedEvent {
	events := make([]storedEvent, 0)
	for _, service := range services {
		for _, container := range service.Containers {
			containerCopy := container
			events = append(events, buildContainerStateEvents(service.TenantID, service.ServerID, service, nil, &containerCopy, container.LastSeenAt)...)
		}
	}
	return sortStoredEvents(events)
}

func buildContainerStateEvents(tenantID, serverID string, service domain.Service, previous *domain.Container, current *domain.Container, observedAt time.Time) []storedEvent {
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}

	events := make([]storedEvent, 0, 3)
	entityName := ""
	containerID := ""
	if current != nil {
		entityName = current.Name
		containerID = current.ID
	} else if previous != nil {
		entityName = previous.Name
		containerID = previous.ID
	}

	appendEvent := func(eventType, message string, timestamp time.Time) {
		if timestamp.IsZero() {
			timestamp = observedAt
		}
		events = append(events, storedEvent{
			TenantID: tenantID,
			ServerID: serverID,
			EventLog: domain.EventLog{
				ServiceID:   service.ID,
				ContainerID: containerID,
				ID:          mustNewUUIDString(),
				Timestamp:   timestamp,
				Type:        eventType,
				Message:     message,
				EntityName:  entityName,
			},
		})
	}

	switch {
	case previous == nil && current != nil:
		if current.Status == "running" {
			appendEvent("start", "Container started", current.LastSeenAt)
		} else {
			appendEvent("stop", "Container stopped", current.LastSeenAt)
		}
		if current.RestartCount > 0 {
			appendEvent("restart", restartMessage(current.RestartCount), current.LastSeenAt)
		}
		if current.Health != "" && current.Health != "healthy" {
			appendEvent("health_change", fmt.Sprintf("Health status changed to %s", current.Health), current.LastSeenAt)
		}
	case previous != nil && current != nil:
		if previous.Status != current.Status {
			if current.Status == "running" {
				appendEvent("start", "Container started", current.LastSeenAt)
			} else {
				appendEvent("stop", "Container stopped", current.LastSeenAt)
			}
		}
		if current.RestartCount > previous.RestartCount {
			appendEvent("restart", restartMessage(current.RestartCount-previous.RestartCount), current.LastSeenAt)
		}
		if previous.Health != current.Health && current.Health != "" {
			appendEvent("health_change", fmt.Sprintf("Health status changed to %s", current.Health), current.LastSeenAt)
		}
	case previous != nil && current == nil:
		if previous.Status == "running" {
			appendEvent("stop", "Container stopped", observedAt)
		}
	}

	return sortStoredEvents(events)
}

func restartMessage(restarts int) string {
	if restarts <= 1 {
		return "Container restarted"
	}
	return fmt.Sprintf("Container restarted %d times", restarts)
}

func sortStoredEvents(events []storedEvent) []storedEvent {
	sort.Slice(events, func(i, j int) bool {
		if events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].ID > events[j].ID
		}
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	return events
}

func cloneEventLogs(events []storedEvent) []domain.EventLog {
	cloned := make([]domain.EventLog, 0, len(events))
	for _, event := range events {
		cloned = append(cloned, event.EventLog)
	}
	return cloned
}
