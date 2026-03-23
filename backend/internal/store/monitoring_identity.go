package store

import (
	"regexp"
	"strings"
)

var uuidStringPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func isUUIDString(value string) bool {
	return uuidStringPattern.MatchString(strings.TrimSpace(strings.ToLower(value)))
}

func monitoringServiceRuntimeKey(composeProject, name string) string {
	composeProject = strings.TrimSpace(composeProject)
	if composeProject != "" {
		return "compose:" + composeProject
	}
	return "name:" + strings.TrimSpace(name)
}

func monitoringContainerRuntimeKey(name string) string {
	return "name:" + strings.TrimSpace(name)
}

func resolveCanonicalMonitoringID(incomingID string) string {
	if isUUIDString(incomingID) {
		return strings.TrimSpace(strings.ToLower(incomingID))
	}
	return mustNewUUIDString()
}
