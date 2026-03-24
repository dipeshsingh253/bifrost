package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrInvalidCredentials = errors.New("invalid credentials")

type SeedData struct {
	Users            []domain.User
	Servers          []domain.Server
	Services         []domain.Service
	Metrics          map[string][]domain.MetricSeries
	ContainerMetrics map[string]domain.ContainerMetricBundle
	Logs             map[string][]domain.LogLine
	Agents           []domain.Agent
}

type MemoryStore struct {
	mu                sync.RWMutex
	users             map[string]domain.User
	usersByID         map[string]domain.User
	disabledUsers     map[string]time.Time
	byToken           map[string]domain.User
	invites           map[string]domain.ViewerInvite
	inviteByToken     map[string]string
	systemOnboardings map[string]domain.SystemOnboarding
	byAPIKey          map[string]domain.Agent
	servers           map[string]domain.Server
	services          map[string]domain.Service
	metrics           map[string][]domain.MetricSeries
	containerMetrics  map[string]domain.ContainerMetricBundle
	logs              map[string][]domain.LogLine
	events            []storedEvent
	agents            map[string]domain.Agent
}

func NewMemoryStore(seed SeedData) *MemoryStore {
	store := &MemoryStore{
		users:             map[string]domain.User{},
		usersByID:         map[string]domain.User{},
		disabledUsers:     map[string]time.Time{},
		byToken:           map[string]domain.User{},
		invites:           map[string]domain.ViewerInvite{},
		inviteByToken:     map[string]string{},
		systemOnboardings: map[string]domain.SystemOnboarding{},
		byAPIKey:          map[string]domain.Agent{},
		servers:           map[string]domain.Server{},
		services:          map[string]domain.Service{},
		metrics:           map[string][]domain.MetricSeries{},
		containerMetrics:  map[string]domain.ContainerMetricBundle{},
		logs:              map[string][]domain.LogLine{},
		events:            []storedEvent{},
		agents:            map[string]domain.Agent{},
	}

	for _, user := range seed.Users {
		store.users[user.Email] = user
		store.usersByID[user.ID] = user
		store.byToken[user.AuthToken] = user
	}

	for _, server := range seed.Servers {
		store.servers[server.ID] = server
	}

	for _, service := range seed.Services {
		store.services[service.ID] = service
	}

	for key, series := range seed.Metrics {
		store.metrics[key] = series
	}

	for key, series := range seed.ContainerMetrics {
		store.containerMetrics[key] = series
	}

	for key, lines := range seed.Logs {
		store.logs[key] = lines
	}

	store.events = append(store.events, seedEventsFromServices(seed.Services)...)

	for _, agent := range seed.Agents {
		store.agents[agent.ID] = agent
		store.byAPIKey[agent.APIKey] = agent
	}

	return store
}

func (s *MemoryStore) BootstrapStatus() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.users) == 0, nil
}

func (s *MemoryStore) BootstrapAdmin(tenantName, name, email, password string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.users) > 0 {
		return domain.User{}, ErrConflict
	}

	tenantID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}
	userID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}

	token := "bootstrap-admin-token"
	user := domain.User{
		ID:        userID,
		TenantID:  tenantID,
		Email:     email,
		Name:      name,
		Password:  password,
		Role:      domain.RoleAdmin,
		AuthToken: token,
	}

	s.users[email] = user
	s.usersByID[user.ID] = user
	s.byToken[token] = user
	return user, nil
}

func (s *MemoryStore) Authenticate(email, password string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[email]
	if !ok || user.Password != password {
		return domain.User{}, ErrNotFound
	}
	if _, disabled := s.disabledUsers[user.ID]; disabled {
		return domain.User{}, ErrNotFound
	}

	if user.AuthToken != "" {
		s.byToken[user.AuthToken] = user
	}

	return user, nil
}

func (s *MemoryStore) UserByToken(token string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.byToken[token]
	if !ok {
		return domain.User{}, ErrNotFound
	}

	return user, nil
}

func (s *MemoryStore) RevokeSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byToken[token]; !ok {
		return ErrNotFound
	}

	delete(s.byToken, token)
	return nil
}

func (s *MemoryStore) UpdateUserName(userID, name string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.usersByID[userID]
	if !ok {
		return domain.User{}, ErrNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return domain.User{}, ErrConflict
	}

	user.Name = name
	s.usersByID[userID] = user
	s.users[user.Email] = user
	if user.AuthToken != "" {
		s.byToken[user.AuthToken] = user
	}

	return user, nil
}

func (s *MemoryStore) ChangeUserPassword(userID, currentPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.usersByID[userID]
	if !ok {
		return ErrNotFound
	}

	if user.Password != currentPassword {
		return ErrInvalidCredentials
	}

	user.Password = strings.TrimSpace(newPassword)
	s.usersByID[userID] = user
	s.users[user.Email] = user
	if user.AuthToken != "" {
		s.byToken[user.AuthToken] = user
	}

	return nil
}

func (s *MemoryStore) ViewerAccess(tenantID string) (domain.ViewerAccess, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	access := domain.ViewerAccess{
		Viewers: []domain.ViewerAccount{},
		Invites: []domain.ViewerInvite{},
	}

	for _, user := range s.usersByID {
		if user.TenantID != tenantID || user.Role != domain.RoleViewer {
			continue
		}
		access.Viewers = append(access.Viewers, domain.ViewerAccount{
			ID:       user.ID,
			TenantID: user.TenantID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
			Status:   viewerStatus(s.disabledUsers, user.ID),
		})
	}

	for _, invite := range s.invites {
		if invite.TenantID != tenantID {
			continue
		}
		invite.Status = inviteLifecycleStatus(invite.AcceptedAt, invite.RevokedAt, invite.ExpiresAt)
		if invite.Status != "pending" {
			continue
		}
		access.Invites = append(access.Invites, invite)
	}

	sort.Slice(access.Viewers, func(i, j int) bool {
		return access.Viewers[i].Email < access.Viewers[j].Email
	})
	sort.Slice(access.Invites, func(i, j int) bool {
		return access.Invites[i].Email < access.Invites[j].Email
	})

	return access, nil
}

func (s *MemoryStore) CreateViewerInvite(tenantID, invitedByUserID, email string) (domain.ViewerInvite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return domain.ViewerInvite{}, ErrConflict
	}

	if existing, ok := s.users[email]; ok && existing.TenantID == tenantID {
		return domain.ViewerInvite{}, ErrConflict
	}

	for _, invite := range s.invites {
		if invite.TenantID == tenantID && strings.EqualFold(invite.Email, email) && invite.AcceptedAt == nil && invite.RevokedAt == nil {
			return domain.ViewerInvite{}, ErrConflict
		}
	}

	now := time.Now().UTC()
	inviteID, err := newUUIDString()
	if err != nil {
		return domain.ViewerInvite{}, err
	}
	invite := domain.ViewerInvite{
		ID:              inviteID,
		TenantID:        tenantID,
		Email:           email,
		Role:            domain.RoleViewer,
		InvitedByUserID: invitedByUserID,
		ExpiresAt:       now.Add(7 * 24 * time.Hour),
		CreatedAt:       now,
		InviteToken:     fmt.Sprintf("invite-%d", now.UnixNano()),
		Status:          "pending",
	}

	s.invites[invite.ID] = invite
	s.inviteByToken[invite.InviteToken] = invite.ID
	return invite, nil
}

func (s *MemoryStore) CreateSystemOnboarding(tenantID, createdByUserID, name, description string) (domain.SystemOnboarding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return domain.SystemOnboarding{}, ErrConflict
	}

	onboardingID, err := newUUIDString()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	serverID, err := newUUIDString()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	apiKey, _, err := newOpaqueToken()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	agentID, err := newUUIDString()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}

	now := time.Now().UTC()
	agent := domain.Agent{
		ID:          agentID,
		TenantID:    tenantID,
		Name:        name,
		APIKey:      apiKey,
		Version:     "pending",
		EnrolledAt:  now,
		ServerName:  name,
		Description: description,
	}
	onboarding := domain.SystemOnboarding{
		ID:              onboardingID,
		TenantID:        tenantID,
		ServerID:        serverID,
		AgentID:         agentID,
		Name:            name,
		Description:     description,
		Status:          "awaiting_connection",
		CreatedByUserID: createdByUserID,
		CreatedAt:       now,
		APIKey:          apiKey,
	}

	s.agents[agent.ID] = agent
	s.byAPIKey[apiKey] = agent
	s.systemOnboardings[onboarding.ID] = onboarding
	return onboarding, nil
}

func (s *MemoryStore) ListSystemOnboardings(tenantID string) ([]domain.SystemOnboarding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.SystemOnboarding, 0, len(s.systemOnboardings))
	for _, onboarding := range s.systemOnboardings {
		if onboarding.TenantID != tenantID {
			continue
		}
		onboarding.APIKey = ""
		result = append(result, onboarding)
	}

	sort.Slice(result, func(i, j int) bool {
		left := result[i]
		right := result[j]

		leftPriority := systemOnboardingStatusPriority(left.Status)
		rightPriority := systemOnboardingStatusPriority(right.Status)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}

		leftTime := left.CreatedAt
		if left.ConnectedAt != nil {
			leftTime = *left.ConnectedAt
		}
		rightTime := right.CreatedAt
		if right.ConnectedAt != nil {
			rightTime = *right.ConnectedAt
		}
		return leftTime.After(rightTime)
	})

	return result, nil
}

func (s *MemoryStore) SystemOnboardingByID(tenantID, onboardingID string) (domain.SystemOnboarding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, onboarding := range s.systemOnboardings {
		if onboarding.TenantID != tenantID {
			continue
		}
		if onboarding.ID == onboardingID {
			return onboarding, nil
		}
	}

	return domain.SystemOnboarding{}, ErrNotFound
}

func (s *MemoryStore) CancelSystemOnboarding(tenantID, onboardingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		onboarding domain.SystemOnboarding
		storedID   string
		found      bool
	)
	for id, candidate := range s.systemOnboardings {
		if candidate.TenantID != tenantID || candidate.ID != onboardingID {
			continue
		}
		onboarding = candidate
		storedID = id
		found = true
		break
	}
	if !found {
		return ErrNotFound
	}
	if onboarding.Status != "awaiting_connection" {
		return ErrConflict
	}

	agent, ok := s.agents[onboarding.AgentID]
	if !ok || agent.TenantID != tenantID {
		return ErrNotFound
	}

	delete(s.byAPIKey, agent.APIKey)
	delete(s.agents, agent.ID)
	delete(s.systemOnboardings, storedID)
	return nil
}

func (s *MemoryStore) ReissueSystemOnboardingCredentials(tenantID, onboardingID string) (domain.SystemOnboarding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		onboarding domain.SystemOnboarding
		storedID   string
		found      bool
	)
	for id, candidate := range s.systemOnboardings {
		if candidate.TenantID != tenantID || candidate.ID != onboardingID {
			continue
		}
		onboarding = candidate
		storedID = id
		found = true
		break
	}
	if !found {
		return domain.SystemOnboarding{}, ErrNotFound
	}
	if onboarding.Status != "awaiting_connection" {
		return domain.SystemOnboarding{}, ErrConflict
	}

	agent, ok := s.agents[onboarding.AgentID]
	if !ok || agent.TenantID != tenantID {
		return domain.SystemOnboarding{}, ErrNotFound
	}

	apiKey, _, err := newOpaqueToken()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}

	delete(s.byAPIKey, agent.APIKey)
	agent.APIKey = apiKey
	onboarding.APIKey = apiKey

	s.agents[agent.ID] = agent
	s.byAPIKey[apiKey] = agent
	s.systemOnboardings[storedID] = onboarding
	return onboarding, nil
}

func (s *MemoryStore) SelfEnrollPendingAgent(agentID, serverID string) (domain.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.agents[agentID]
	if !ok {
		return domain.Agent{}, ErrNotFound
	}
	if agent.Version != "pending" {
		return domain.Agent{}, ErrConflict
	}

	var onboarding domain.SystemOnboarding
	found := false
	for _, candidate := range s.systemOnboardings {
		if candidate.AgentID != agentID {
			continue
		}
		onboarding = candidate
		found = true
		break
	}
	if !found {
		return domain.Agent{}, ErrNotFound
	}
	if onboarding.Status != "awaiting_connection" || onboarding.ServerID != serverID {
		return domain.Agent{}, ErrConflict
	}

	newAPIKey, _, err := newOpaqueToken()
	if err != nil {
		return domain.Agent{}, err
	}

	delete(s.byAPIKey, agent.APIKey)
	agent.APIKey = newAPIKey
	agent.ServerID = onboarding.ServerID
	agent.Version = "enrolled"
	agent.LastSeenAt = time.Now().UTC()
	if agent.EnrolledAt.IsZero() {
		agent.EnrolledAt = agent.LastSeenAt
	}
	s.agents[agent.ID] = agent
	s.byAPIKey[newAPIKey] = agent
	return agent, nil
}

func (s *MemoryStore) InviteByToken(token string) (domain.ViewerInvite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inviteID, ok := s.inviteByToken[token]
	if !ok {
		return domain.ViewerInvite{}, ErrNotFound
	}
	invite, ok := s.invites[inviteID]
	if !ok {
		return domain.ViewerInvite{}, ErrNotFound
	}
	if inviteLifecycleStatus(invite.AcceptedAt, invite.RevokedAt, invite.ExpiresAt) != "pending" {
		return domain.ViewerInvite{}, ErrConflict
	}
	return invite, nil
}

func (s *MemoryStore) AcceptViewerInvite(token, name, password string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inviteID, ok := s.inviteByToken[token]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	invite, ok := s.invites[inviteID]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	if invite.RevokedAt != nil || invite.AcceptedAt != nil || invite.ExpiresAt.Before(time.Now().UTC()) {
		return domain.User{}, ErrConflict
	}
	if _, ok := s.users[invite.Email]; ok {
		return domain.User{}, ErrConflict
	}

	userID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}
	user := domain.User{
		ID:        userID,
		TenantID:  invite.TenantID,
		Email:     invite.Email,
		Name:      name,
		Password:  password,
		Role:      domain.RoleViewer,
		AuthToken: "viewer-session-" + strings.ReplaceAll(invite.Email, "@", "-"),
	}
	now := time.Now().UTC()
	invite.AcceptedAt = &now
	invite.Status = "accepted"
	s.invites[invite.ID] = invite
	s.users[user.Email] = user
	s.usersByID[user.ID] = user
	s.byToken[user.AuthToken] = user
	return user, nil
}

func (s *MemoryStore) RevokeViewerInvite(tenantID, inviteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	invite, ok := s.invites[inviteID]
	if !ok || invite.TenantID != tenantID {
		return ErrNotFound
	}
	if invite.RevokedAt != nil || invite.AcceptedAt != nil {
		return ErrConflict
	}
	now := time.Now().UTC()
	invite.RevokedAt = &now
	invite.Status = "revoked"
	s.invites[inviteID] = invite
	return nil
}

func (s *MemoryStore) DisableViewer(tenantID, viewerUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.usersByID[viewerUserID]
	if !ok || user.TenantID != tenantID || user.Role != domain.RoleViewer {
		return ErrNotFound
	}

	for token, tokenUser := range s.byToken {
		if tokenUser.ID == viewerUserID {
			delete(s.byToken, token)
		}
	}
	s.disabledUsers[viewerUserID] = time.Now().UTC()
	return nil
}

func (s *MemoryStore) DeleteViewer(tenantID, viewerUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.usersByID[viewerUserID]
	if !ok || user.TenantID != tenantID || user.Role != domain.RoleViewer {
		return ErrNotFound
	}

	delete(s.usersByID, viewerUserID)
	delete(s.users, user.Email)
	delete(s.disabledUsers, viewerUserID)
	for token, tokenUser := range s.byToken {
		if tokenUser.ID == viewerUserID {
			delete(s.byToken, token)
		}
	}
	return nil
}

func (s *MemoryStore) TenantSummary(tenantID string) (domain.TenantSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := domain.TenantSummary{
		TenantID:   tenantID,
		TenantName: "Bifrost",
	}

	foundTenant := false
	for _, user := range s.users {
		if user.TenantID != tenantID {
			continue
		}
		foundTenant = true
		switch user.Role {
		case domain.RoleViewer:
			summary.ViewerCount++
		default:
			summary.AdminCount++
		}
	}

	if !foundTenant {
		return domain.TenantSummary{}, ErrNotFound
	}

	return summary, nil
}

func (s *MemoryStore) ListServers(tenantID string) []domain.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()

	servers := make([]domain.Server, 0)
	for _, server := range s.servers {
		if server.TenantID == tenantID {
			servers = append(servers, server)
		}
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers
}

func (s *MemoryStore) ServerByID(tenantID, serverID string) (domain.Server, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, server := range s.servers {
		if server.TenantID != tenantID {
			continue
		}
		if server.ID == serverID {
			return server, nil
		}
	}

	return domain.Server{}, ErrNotFound
}

func (s *MemoryStore) ServicesByServer(tenantID, serverID string) []domain.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make([]domain.Service, 0)
	for _, service := range s.services {
		if service.TenantID == tenantID && service.ServerID == serverID {
			services = append(services, service)
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

func (s *MemoryStore) ServiceByID(tenantID, serviceID string) (domain.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, service := range s.services {
		if service.TenantID != tenantID {
			continue
		}
		if service.ID == serviceID {
			return cloneService(service), nil
		}
	}

	return domain.Service{}, ErrNotFound
}

func (s *MemoryStore) ProjectByID(tenantID, serverID, projectID string) (domain.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, service := range s.services {
		if service.TenantID != tenantID || service.ServerID != serverID || service.ComposeProject == "" {
			continue
		}
		if service.ID == projectID {
			return cloneService(service), nil
		}
	}

	return domain.Service{}, ErrNotFound
}

func (s *MemoryStore) ProjectsByServer(tenantID, serverID string) []domain.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]domain.Service, 0)
	for _, service := range s.services {
		if service.TenantID == tenantID && service.ServerID == serverID && service.ComposeProject != "" {
			projects = append(projects, cloneService(service))
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects
}

func (s *MemoryStore) StandaloneContainersByServer(tenantID, serverID string) []domain.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()

	containers := make([]domain.Container, 0)
	for _, service := range s.services {
		if service.TenantID != tenantID || service.ServerID != serverID || service.ComposeProject != "" {
			continue
		}

		for _, container := range service.Containers {
			containerCopy := container
			containerCopy.Ports = append([]string(nil), container.Ports...)
			containers = append(containers, containerCopy)
		}
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers
}

func (s *MemoryStore) MetricsByServer(serverID string) []domain.MetricSeries {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneMetricSeries(s.metrics[serverID])
}

func (s *MemoryStore) ContainerMetricsByServer(serverID string) domain.ContainerMetricBundle {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneContainerMetricBundle(s.containerMetrics[serverID])
}

func (s *MemoryStore) LogsByService(serviceID string) []domain.LogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lines := append([]domain.LogLine(nil), s.logs[serviceID]...)
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Timestamp.Before(lines[j].Timestamp)
	})
	return lines
}

func (s *MemoryStore) LogsByContainer(serviceID, containerID string) []domain.LogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lines := make([]domain.LogLine, 0)
	for _, line := range s.logs[serviceID] {
		if line.ContainerID == containerID {
			lines = append(lines, line)
		}
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Timestamp.Before(lines[j].Timestamp)
	})
	return lines
}

func (s *MemoryStore) ServerBundle(tenantID, serverID string) (domain.ServerBundle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	server, ok := s.servers[serverID]
	if !ok || server.TenantID != tenantID {
		return domain.ServerBundle{}, ErrNotFound
	}

	services := make([]domain.Service, 0)
	for _, service := range s.services {
		if service.TenantID == tenantID && service.ServerID == serverID {
			services = append(services, cloneService(service))
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	containerIDs := currentContainerIDs(services)

	return domain.ServerBundle{
		Server:           server,
		Services:         services,
		Metrics:          cloneMetricSeries(s.metrics[serverID]),
		ContainerMetrics: filterContainerMetricBundleByKeys(cloneContainerMetricBundle(s.containerMetrics[serverID]), containerIDs),
	}, nil
}

func (s *MemoryStore) ContainerByID(tenantID, serverID, containerID string) (domain.Container, domain.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, service := range s.services {
		if service.TenantID != tenantID || service.ServerID != serverID {
			continue
		}

		for _, container := range service.Containers {
			if container.ID == containerID {
				containerCopy := container
				containerCopy.Ports = append([]string(nil), container.Ports...)
				return containerCopy, cloneService(service), nil
			}
		}
	}

	return domain.Container{}, domain.Service{}, ErrNotFound
}

func (s *MemoryStore) ProjectMetrics(tenantID, serverID, projectID string) (domain.ContainerMetricBundle, error) {
	project, err := s.ProjectByID(tenantID, serverID, projectID)
	if err != nil {
		return domain.ContainerMetricBundle{}, err
	}

	containerIDs := make([]string, 0, len(project.Containers))
	for _, container := range project.Containers {
		containerIDs = append(containerIDs, container.ID)
	}

	return filterContainerMetricBundleByKeys(s.ContainerMetricsByServer(serverID), containerIDs), nil
}

func (s *MemoryStore) ContainerMetrics(tenantID, serverID, containerID string) (domain.ContainerMetricHistory, domain.Container, domain.Service, error) {
	container, service, err := s.ContainerByID(tenantID, serverID, containerID)
	if err != nil {
		return domain.ContainerMetricHistory{}, domain.Container{}, domain.Service{}, err
	}

	bundle := s.ContainerMetricsByServer(serverID)
	return containerMetricHistoryByKey(bundle, container.ID), container, service, nil
}

func (s *MemoryStore) ProjectEvents(tenantID, serverID, projectID string) ([]domain.EventLog, domain.Service, error) {
	project, err := s.ProjectByID(tenantID, serverID, projectID)
	if err != nil {
		return nil, domain.Service{}, err
	}

	return s.projectEvents(project.ID), project, nil
}

func (s *MemoryStore) ContainerEvents(tenantID, serverID, containerID string) ([]domain.EventLog, domain.Container, domain.Service, error) {
	container, service, err := s.ContainerByID(tenantID, serverID, containerID)
	if err != nil {
		return nil, domain.Container{}, domain.Service{}, err
	}

	return s.containerEvents(container.ID), container, service, nil
}

func (s *MemoryStore) ContainerEnv(tenantID, serverID, containerID string) (map[string]string, domain.Container, domain.Service, error) {
	container, service, err := s.ContainerByID(tenantID, serverID, containerID)
	if err != nil {
		return nil, domain.Container{}, domain.Service{}, err
	}

	return deriveContainerEnv(container, service), container, service, nil
}

func (s *MemoryStore) EnrollAgent(agent domain.Agent) (domain.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	agent.EnrolledAt = now
	agent.LastSeenAt = now
	s.agents[agent.ID] = agent
	s.byAPIKey[agent.APIKey] = agent
	return agent, nil
}

func (s *MemoryStore) AgentByAPIKey(apiKey string) (domain.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.byAPIKey[apiKey]
	if !ok {
		return domain.Agent{}, ErrNotFound
	}

	return agent, nil
}

func (s *MemoryStore) UpdateAgentLastSeen(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.agents[agentID]
	if !ok {
		return ErrNotFound
	}

	agent.LastSeenAt = time.Now().UTC()
	s.agents[agentID] = agent
	return nil
}

func (s *MemoryStore) Ingest(payload domain.IngestPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.agents[payload.AgentID]
	if !ok {
		return ErrNotFound
	}

	server, ok := s.servers[payload.Server.ID]
	if !ok {
		server = domain.Server{
			ID:       payload.Server.ID,
			TenantID: agent.TenantID,
		}
	}

	server.Name = payload.Server.Name
	server.Hostname = payload.Server.Hostname
	server.PublicIP = payload.Server.PublicIP
	server.AgentVersion = payload.Server.AgentVersion
	server.Status = normalizeServerStatus(payload.Server.Status)
	server.LastSeenAt = payload.Server.CollectedAt
	server.UptimeSeconds = payload.Server.UptimeSeconds
	server.CPUUsagePct = payload.Server.CPUUsagePct
	server.MemoryUsagePct = payload.Server.MemoryUsagePct
	server.DiskUsagePct = payload.Server.DiskUsagePct
	server.NetworkRXMB = payload.Server.NetworkRXMB
	server.NetworkTXMB = payload.Server.NetworkTXMB
	server.LoadAverage = payload.Server.LoadAverage
	server.OS = payload.Server.OS
	server.Kernel = payload.Server.Kernel
	server.CPUModel = payload.Server.CPUModel
	server.CPUCores = payload.Server.CPUCores
	server.CPUThreads = payload.Server.CPUThreads
	server.TotalMemoryGB = payload.Server.TotalMemoryGB
	server.TotalDiskGB = payload.Server.TotalDiskGB
	s.servers[server.ID] = server

	agent.ServerID = server.ID
	if server.Name != "" {
		agent.ServerName = server.Name
	}
	if server.Hostname != "" {
		agent.Hostname = server.Hostname
	}
	agent.LastSeenAt = payload.Server.CollectedAt
	if agent.LastSeenAt.IsZero() {
		agent.LastSeenAt = time.Now().UTC()
	}
	s.agents[agent.ID] = agent

	observedAt := payload.Server.CollectedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	previousContainers := map[string]domain.Container{}
	previousServices := map[string]domain.Service{}
	existingServicesByRuntimeKey := map[string]domain.Service{}
	existingContainersByServiceRuntimeKey := map[string]map[string]domain.Container{}
	for _, existingService := range s.services {
		if existingService.ServerID != server.ID {
			continue
		}
		serviceCopy := cloneService(existingService)
		existingServicesByRuntimeKey[monitoringServiceRuntimeKey(serviceCopy.ComposeProject, serviceCopy.Name)] = serviceCopy
		containersByRuntimeKey := make(map[string]domain.Container, len(serviceCopy.Containers))
		for _, container := range serviceCopy.Containers {
			previousContainers[container.ID] = container
			previousServices[container.ID] = serviceCopy
			containersByRuntimeKey[monitoringContainerRuntimeKey(container.Name)] = container
		}
		existingContainersByServiceRuntimeKey[monitoringServiceRuntimeKey(serviceCopy.ComposeProject, serviceCopy.Name)] = containersByRuntimeKey
	}

	for id, onboarding := range s.systemOnboardings {
		if onboarding.AgentID != agent.ID {
			continue
		}
		if onboarding.Status != "connected" {
			connectedAt := payload.Server.CollectedAt
			if connectedAt.IsZero() {
				connectedAt = time.Now().UTC()
			}
			onboarding.Status = "connected"
			onboarding.ConnectedAt = &connectedAt
			s.systemOnboardings[id] = onboarding
		}
		break
	}

	incomingServiceIDs := make(map[string]struct{}, len(payload.Server.Services))
	seenContainerIDs := make(map[string]struct{})
	serviceIDMap := make(map[string]string, len(payload.Server.Services))
	containerIDMap := make(map[string]string)
	canonicalServices := make([]domain.ServiceSnapshot, 0, len(payload.Server.Services))

	for _, snapshot := range payload.Server.Services {
		runtimeServiceKey := monitoringServiceRuntimeKey(snapshot.ComposeProject, snapshot.Name)
		canonicalServiceID := resolveCanonicalMonitoringID(snapshot.ID)
		if existingService, ok := existingServicesByRuntimeKey[runtimeServiceKey]; ok {
			canonicalServiceID = existingService.ID
		}
		serviceIDMap[snapshot.ID] = canonicalServiceID
		incomingServiceIDs[canonicalServiceID] = struct{}{}

		canonicalSnapshot := snapshot
		canonicalSnapshot.ID = canonicalServiceID
		canonicalSnapshot.Containers = make([]domain.ContainerSnapshot, 0, len(snapshot.Containers))
		service := domain.Service{
			ID:             canonicalServiceID,
			TenantID:       server.TenantID,
			ServerID:       server.ID,
			Name:           snapshot.Name,
			ComposeProject: snapshot.ComposeProject,
			Status:         snapshot.Status,
			ContainerCount: len(snapshot.Containers),
			PublishedPorts: snapshot.PublishedPorts,
			Containers:     make([]domain.Container, 0, len(snapshot.Containers)),
		}
		previousContainersByRuntimeKey := existingContainersByServiceRuntimeKey[runtimeServiceKey]

		restarts := 0
		serviceStatus := normalizeServiceStatus(snapshot.Status)
		lastSeenAt := payload.Server.CollectedAt
		for _, containerSnapshot := range snapshot.Containers {
			canonicalContainerID := resolveCanonicalMonitoringID(containerSnapshot.ID)
			if previousContainer, ok := previousContainersByRuntimeKey[monitoringContainerRuntimeKey(containerSnapshot.Name)]; ok {
				canonicalContainerID = previousContainer.ID
			}
			containerIDMap[containerSnapshot.ID] = canonicalContainerID

			canonicalContainerSnapshot := containerSnapshot
			canonicalContainerSnapshot.ID = canonicalContainerID
			canonicalSnapshot.Containers = append(canonicalSnapshot.Containers, canonicalContainerSnapshot)

			restarts += containerSnapshot.RestartCount
			container := domain.Container{
				ID:           canonicalContainerID,
				ServiceID:    canonicalServiceID,
				Name:         containerSnapshot.Name,
				Image:        containerSnapshot.Image,
				Status:       normalizeContainerStatus(containerSnapshot.Status),
				Health:       normalizeHealth(containerSnapshot.Health),
				CPUUsagePct:  containerSnapshot.CPUUsagePct,
				MemoryMB:     containerSnapshot.MemoryMB,
				NetworkMB:    containerSnapshot.NetworkMB,
				RestartCount: containerSnapshot.RestartCount,
				Uptime:       containerSnapshot.Uptime,
				Ports:        containerSnapshot.Ports,
				Command:      containerSnapshot.Command,
				LastSeenAt:   containerSnapshot.LastSeenAt,
			}
			if container.LastSeenAt.IsZero() {
				container.LastSeenAt = lastSeenAt
			}
			service.Containers = append(service.Containers, container)
			serviceStatus = rollupServiceStatus(serviceStatus, container.Status, container.Health)

			var previous *domain.Container
			if prior, ok := previousContainers[container.ID]; ok {
				priorCopy := prior
				previous = &priorCopy
			}
			s.events = append(s.events, buildContainerStateEvents(agent.TenantID, server.ID, service, previous, &container, observedAt)...)
			seenContainerIDs[container.ID] = struct{}{}
		}

		service.RestartCount = restarts
		service.Status = serviceStatus
		s.services[service.ID] = service
		canonicalServices = append(canonicalServices, canonicalSnapshot)
	}

	for containerID, previous := range previousContainers {
		if _, ok := seenContainerIDs[containerID]; ok {
			continue
		}
		previousCopy := previous
		s.events = append(s.events, buildContainerStateEvents(agent.TenantID, server.ID, previousServices[containerID], &previousCopy, nil, observedAt)...)
	}

	s.pruneServicesForServer(server.ID, incomingServiceIDs)

	for _, metric := range payload.Metrics {
		serverMetricID := metric.ServerID
		if serverMetricID == "" {
			serverMetricID = server.ID
		}
		s.metrics[serverMetricID] = mergeMetricSeries(s.metrics[serverMetricID], domain.MetricSeries{
			Key:    metric.Key,
			Unit:   metric.Unit,
			Points: metric.Points,
		})
	}

	s.containerMetrics[server.ID] = appendContainerMetrics(s.containerMetrics[server.ID], payload.Server.CollectedAt, canonicalServices)

	for _, logLine := range payload.Logs {
		serviceID := serviceIDMap[logLine.ServiceID]
		containerID := containerIDMap[logLine.ContainerID]
		containerName, serviceTag := lookupLogContext(canonicalServices, serviceID, containerID)
		line := domain.LogLine{
			ID:            mustNewUUIDString(),
			ServerID:      server.ID,
			ServiceID:     serviceID,
			ContainerID:   containerID,
			ContainerName: containerName,
			ServiceTag:    serviceTag,
			Level:         logLine.Level,
			Message:       logLine.Message,
			Timestamp:     logLine.Timestamp,
		}
		s.logs[line.ServiceID] = append(s.logs[line.ServiceID], line)
		if service, ok := s.services[line.ServiceID]; ok && (service.LastLogTimestamp.IsZero() || line.Timestamp.After(service.LastLogTimestamp)) {
			service.LastLogTimestamp = line.Timestamp
			s.services[line.ServiceID] = service
		}
	}

	return nil
}

func (s *MemoryStore) projectEvents(serviceID string) []domain.EventLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]storedEvent, 0)
	for _, event := range s.events {
		if event.ServiceID == serviceID {
			filtered = append(filtered, event)
		}
	}
	return cloneEventLogs(sortStoredEvents(filtered))
}

func (s *MemoryStore) containerEvents(containerID string) []domain.EventLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]storedEvent, 0)
	for _, event := range s.events {
		if event.ContainerID == containerID {
			filtered = append(filtered, event)
		}
	}
	return cloneEventLogs(sortStoredEvents(filtered))
}

func mergeMetricSeries(existing []domain.MetricSeries, incoming domain.MetricSeries) []domain.MetricSeries {
	for index := range existing {
		if existing[index].Key == incoming.Key {
			existing[index].Points = append(existing[index].Points, incoming.Points...)
			if len(existing[index].Points) > 30 {
				existing[index].Points = existing[index].Points[len(existing[index].Points)-30:]
			}
			return existing
		}
	}

	return append(existing, incoming)
}

func cloneService(service domain.Service) domain.Service {
	copyService := service
	copyService.PublishedPorts = append([]string(nil), service.PublishedPorts...)
	copyService.Containers = make([]domain.Container, 0, len(service.Containers))
	for _, container := range service.Containers {
		containerCopy := container
		containerCopy.Ports = append([]string(nil), container.Ports...)
		copyService.Containers = append(copyService.Containers, containerCopy)
	}
	return copyService
}

func cloneMetricSeries(series []domain.MetricSeries) []domain.MetricSeries {
	cloned := make([]domain.MetricSeries, 0, len(series))
	for _, item := range series {
		cloned = append(cloned, domain.MetricSeries{
			Key:    item.Key,
			Unit:   item.Unit,
			Points: append([]domain.MetricPoint(nil), item.Points...),
		})
	}
	return cloned
}

func cloneContainerMetricPoints(points []domain.ContainerMetricPoint) []domain.ContainerMetricPoint {
	cloned := make([]domain.ContainerMetricPoint, 0, len(points))
	for _, point := range points {
		values := make(map[string]float64, len(point.Values))
		for key, value := range point.Values {
			values[key] = value
		}
		cloned = append(cloned, domain.ContainerMetricPoint{
			Timestamp: point.Timestamp,
			Values:    values,
		})
	}
	return cloned
}

func cloneContainerMetricBundle(bundle domain.ContainerMetricBundle) domain.ContainerMetricBundle {
	return domain.ContainerMetricBundle{
		CPU:     cloneContainerMetricPoints(bundle.CPU),
		Memory:  cloneContainerMetricPoints(bundle.Memory),
		Network: cloneContainerMetricPoints(bundle.Network),
	}
}

func appendContainerMetrics(existing domain.ContainerMetricBundle, timestamp time.Time, services []domain.ServiceSnapshot) domain.ContainerMetricBundle {
	cpuValues := map[string]float64{}
	memoryValues := map[string]float64{}
	networkValues := map[string]float64{}

	for _, service := range services {
		for _, container := range service.Containers {
			cpuValues[container.ID] = container.CPUUsagePct
			memoryValues[container.ID] = container.MemoryMB
			networkValues[container.ID] = container.NetworkMB
		}
	}

	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	existing.CPU = appendMetricPoint(existing.CPU, timestamp, cpuValues)
	existing.Memory = appendMetricPoint(existing.Memory, timestamp, memoryValues)
	existing.Network = appendMetricPoint(existing.Network, timestamp, networkValues)
	return existing
}

func appendMetricPoint(existing []domain.ContainerMetricPoint, timestamp time.Time, values map[string]float64) []domain.ContainerMetricPoint {
	pointValues := make(map[string]float64, len(values))
	for key, value := range values {
		pointValues[key] = value
	}

	existing = append(existing, domain.ContainerMetricPoint{
		Timestamp: timestamp,
		Values:    pointValues,
	})
	if len(existing) > 30 {
		existing = existing[len(existing)-30:]
	}
	return existing
}

func currentContainerIDs(services []domain.Service) []string {
	ids := make([]string, 0)
	for _, service := range services {
		for _, container := range service.Containers {
			ids = append(ids, container.ID)
		}
	}
	return ids
}

func (s *MemoryStore) pruneServicesForServer(serverID string, keep map[string]struct{}) {
	for serviceID, service := range s.services {
		if service.ServerID != serverID {
			continue
		}
		if _, ok := keep[serviceID]; ok {
			continue
		}
		delete(s.services, serviceID)
		delete(s.logs, serviceID)
	}
}

func filterContainerMetricBundleByKeys(bundle domain.ContainerMetricBundle, keys []string) domain.ContainerMetricBundle {
	allowed := map[string]struct{}{}
	for _, key := range keys {
		allowed[key] = struct{}{}
	}

	return domain.ContainerMetricBundle{
		CPU:     filterContainerMetricPoints(bundle.CPU, allowed),
		Memory:  filterContainerMetricPoints(bundle.Memory, allowed),
		Network: filterContainerMetricPoints(bundle.Network, allowed),
	}
}

func filterContainerMetricPoints(points []domain.ContainerMetricPoint, allowed map[string]struct{}) []domain.ContainerMetricPoint {
	filtered := make([]domain.ContainerMetricPoint, 0, len(points))
	for _, point := range points {
		values := map[string]float64{}
		for key, value := range point.Values {
			if _, ok := allowed[key]; ok {
				values[key] = value
			}
		}
		filtered = append(filtered, domain.ContainerMetricPoint{
			Timestamp: point.Timestamp,
			Values:    values,
		})
	}
	return filtered
}

func containerMetricHistoryByKey(bundle domain.ContainerMetricBundle, key string) domain.ContainerMetricHistory {
	return domain.ContainerMetricHistory{
		CPU:     metricSeriesForContainer(bundle.CPU, key),
		Memory:  metricSeriesForContainer(bundle.Memory, key),
		Network: metricSeriesForContainer(bundle.Network, key),
	}
}

func metricSeriesForContainer(points []domain.ContainerMetricPoint, key string) []domain.MetricPoint {
	series := make([]domain.MetricPoint, 0, len(points))
	for _, point := range points {
		series = append(series, domain.MetricPoint{
			Timestamp: point.Timestamp,
			Value:     point.Values[key],
		})
	}
	return series
}

func deriveProjectEvents(project domain.Service, logs []domain.LogLine) []domain.EventLog {
	events := make([]domain.EventLog, 0)
	for _, container := range project.Containers {
		events = append(events, deriveContainerEvents(container, project, filterLogsByContainer(logs, container.ID))...)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	return limitEvents(events, 20)
}

func deriveContainerEvents(container domain.Container, service domain.Service, logs []domain.LogLine) []domain.EventLog {
	events := make([]domain.EventLog, 0)

	if container.Status == "running" {
		events = append(events, domain.EventLog{
			ID:         mustNewUUIDString(),
			Timestamp:  container.LastSeenAt,
			Type:       "start",
			Message:    "Container started",
			EntityName: container.Name,
		})
	} else {
		events = append(events, domain.EventLog{
			ID:         mustNewUUIDString(),
			Timestamp:  container.LastSeenAt,
			Type:       "stop",
			Message:    "Container stopped",
			EntityName: container.Name,
		})
	}

	if container.RestartCount > 0 {
		events = append(events, domain.EventLog{
			ID:         mustNewUUIDString(),
			Timestamp:  container.LastSeenAt.Add(-1 * time.Minute),
			Type:       "restart",
			Message:    fmt.Sprintf("Container restarted %d times", container.RestartCount),
			EntityName: container.Name,
		})
	}

	if container.Health == "unhealthy" {
		events = append(events, domain.EventLog{
			ID:         mustNewUUIDString(),
			Timestamp:  container.LastSeenAt.Add(-30 * time.Second),
			Type:       "health_change",
			Message:    "Health status changed to unhealthy",
			EntityName: container.Name,
		})
	}

	for _, line := range logs {
		if strings.EqualFold(line.Level, "error") {
			events = append(events, domain.EventLog{
				ID:         mustNewUUIDString(),
				Timestamp:  line.Timestamp,
				Type:       "crash",
				Message:    line.Message,
				EntityName: container.Name,
			})
			break
		}
	}

	if len(events) == 0 && service.Name != "" {
		events = append(events, domain.EventLog{
			ID:         mustNewUUIDString(),
			Timestamp:  container.LastSeenAt,
			Type:       "start",
			Message:    "Container observed by backend",
			EntityName: container.Name,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	return limitEvents(events, 10)
}

func deriveContainerEnv(container domain.Container, service domain.Service) map[string]string {
	port := "8080"
	if len(container.Ports) > 0 {
		port = strings.Split(container.Ports[0], ":")[0]
	}

	env := map[string]string{
		"CONTAINER_NAME": container.Name,
		"IMAGE":          container.Image,
		"LOG_LEVEL":      "info",
		"NODE_ENV":       "production",
		"PORT":           port,
		"SERVICE_NAME":   service.Name,
		"TZ":             "UTC",
	}
	if service.ComposeProject != "" {
		env["COMPOSE_PROJECT"] = service.ComposeProject
	}

	return env
}

func filterLogsByContainer(lines []domain.LogLine, containerID string) []domain.LogLine {
	filtered := make([]domain.LogLine, 0)
	for _, line := range lines {
		if line.ContainerID == containerID {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func FilterLogs(lines []domain.LogLine, search string, limit int) []domain.LogLine {
	filtered := make([]domain.LogLine, 0, len(lines))
	search = strings.TrimSpace(strings.ToLower(search))
	for _, line := range lines {
		if search != "" {
			if !strings.Contains(strings.ToLower(line.Message), search) &&
				!strings.Contains(strings.ToLower(line.ContainerName), search) &&
				!strings.Contains(strings.ToLower(line.ServiceTag), search) {
				continue
			}
		}
		filtered = append(filtered, line)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered
}

func viewerStatus(disabled map[string]time.Time, userID string) string {
	if _, ok := disabled[userID]; ok {
		return "disabled"
	}
	return "active"
}

func inviteLifecycleStatus(acceptedAt, revokedAt *time.Time, expiresAt time.Time) string {
	switch {
	case acceptedAt != nil:
		return "accepted"
	case revokedAt != nil:
		return "revoked"
	case !expiresAt.IsZero() && expiresAt.Before(time.Now().UTC()):
		return "expired"
	default:
		return "pending"
	}
}

func systemOnboardingStatusPriority(status string) int {
	switch status {
	case "awaiting_connection":
		return 0
	case "connected":
		return 1
	default:
		return 2
	}
}

func limitEvents(events []domain.EventLog, limit int) []domain.EventLog {
	if limit > 0 && len(events) > limit {
		return events[:limit]
	}
	return events
}

func normalizeServerStatus(status string) string {
	switch status {
	case "healthy", "running", "up":
		return "up"
	case "down", "offline", "stopped":
		return "down"
	default:
		return status
	}
}

func normalizeServiceStatus(status string) string {
	switch status {
	case "healthy", "running", "up":
		return "running"
	case "degraded", "unhealthy":
		return "degraded"
	case "stopped", "down", "exited":
		return "stopped"
	case "":
		return "running"
	default:
		return status
	}
}

func normalizeContainerStatus(status string) string {
	switch status {
	case "healthy", "running", "up":
		return "running"
	case "exited", "stopped", "dead", "down":
		return "exited"
	default:
		return status
	}
}

func normalizeHealth(health string) string {
	switch health {
	case "", "unknown":
		return "unknown"
	case "degraded":
		return "unhealthy"
	default:
		return health
	}
}

func rollupServiceStatus(current, containerStatus, health string) string {
	if containerStatus != "running" {
		return "stopped"
	}
	if health == "unhealthy" && current == "running" {
		return "degraded"
	}
	return current
}

func lookupLogContext(services []domain.ServiceSnapshot, serviceID, containerID string) (string, string) {
	for _, service := range services {
		if service.ID != serviceID {
			continue
		}

		for _, container := range service.Containers {
			if container.ID == containerID {
				return container.Name, serviceTag(service, container.Name)
			}
		}
	}

	return "", ""
}

func serviceTag(service domain.ServiceSnapshot, containerName string) string {
	if service.ComposeProject == "" || service.Name == "" {
		return containerName
	}

	prefix := service.Name + "-"
	if len(containerName) > len(prefix) && containerName[:len(prefix)] == prefix {
		remainder := containerName[len(prefix):]
		for index := 0; index < len(remainder); index++ {
			if remainder[index] == '-' {
				return remainder[:index]
			}
		}
		return remainder
	}

	return service.Name
}
