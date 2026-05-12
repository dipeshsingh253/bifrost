package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStoreFromDB(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return NewPostgresStoreFromDB(db), nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) BootstrapStatus() (bool, error) {
	ctx := context.Background()

	var userCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return false, err
	}

	return userCount == 0, nil
}

func (s *PostgresStore) EnsureSeedData(seed SeedData) error {
	ctx := context.Background()

	var tenantCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&tenantCount); err != nil {
		return err
	}
	if tenantCount > 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, user := range seed.Users {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		role := mvpUserRole(user.Role)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tenants (id, name, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (id) DO NOTHING
		`, user.TenantID, "Bifrost Demo"); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO users (id, tenant_id, email, password_hash, name, role, created_at, is_active, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), TRUE, NOW())
			ON CONFLICT (id) DO NOTHING
		`, user.ID, user.TenantID, user.Email, string(hashedPassword), user.Name, role); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tenant_memberships (id, tenant_id, user_id, role, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())
			ON CONFLICT (tenant_id, user_id) DO NOTHING
		`, mustNewUUIDString(), user.TenantID, user.ID, role); err != nil {
			return err
		}

		if user.AuthToken != "" {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO user_sessions (
					id, tenant_id, user_id, session_token_hash, expires_at, last_seen_at, created_at, user_agent, ip_address
				)
				VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), '', '')
				ON CONFLICT (session_token_hash) DO NOTHING
			`, mustNewUUIDString(), user.TenantID, user.ID, hashSecret(user.AuthToken), time.Now().UTC().Add(365*24*time.Hour)); err != nil {
				return err
			}
		}
	}

	serverByID := make(map[string]domain.Server, len(seed.Servers))
	for _, server := range seed.Servers {
		serverByID[server.ID] = server
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO servers (
				id, tenant_id, name, hostname, public_ip, agent_version, status, last_seen_at, uptime_seconds,
				cpu_usage_pct, memory_usage_pct, disk_usage_pct, network_rx_mb, network_tx_mb, load_average,
				os, kernel, cpu_model, cpu_cores, cpu_threads, total_memory_gb, total_disk_gb, created_at, updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15,
				$16, $17, $18, $19, $20, $21, $22, NOW(), NOW()
			)
			ON CONFLICT (id) DO NOTHING
		`,
			server.ID, server.TenantID, server.Name, server.Hostname, server.PublicIP, server.AgentVersion, server.Status, server.LastSeenAt, server.UptimeSeconds,
			server.CPUUsagePct, server.MemoryUsagePct, server.DiskUsagePct, server.NetworkRXMB, server.NetworkTXMB, server.LoadAverage,
			server.OS, server.Kernel, server.CPUModel, server.CPUCores, server.CPUThreads, server.TotalMemoryGB, server.TotalDiskGB,
		); err != nil {
			return err
		}
	}

	containerIDToServiceID := map[string]string{}
	for _, service := range seed.Services {
		ports, err := marshalStrings(service.PublishedPorts)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO service_groups (
				id, tenant_id, server_id, name, compose_project, status, container_count, restart_count,
				published_ports, updated_at, last_log_timestamp, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10, NOW())
			ON CONFLICT (id) DO NOTHING
		`, service.ID, service.TenantID, service.ServerID, service.Name, service.ComposeProject, service.Status, service.ContainerCount, service.RestartCount, ports, service.LastLogTimestamp); err != nil {
			return err
		}

		for _, container := range service.Containers {
			containerIDToServiceID[container.ID] = service.ID
			containerPorts, err := marshalStrings(container.Ports)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO containers (
					id, service_group_id, name, image, status, health, cpu_usage_pct, memory_mb, network_mb,
					restart_count, uptime, ports, command, last_seen_at, created_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
				ON CONFLICT (id) DO NOTHING
			`, container.ID, service.ID, container.Name, container.Image, container.Status, container.Health, container.CPUUsagePct, container.MemoryMB, container.NetworkMB, container.RestartCount, container.Uptime, containerPorts, container.Command, container.LastSeenAt); err != nil {
				return err
			}
		}
	}

	for serverID, metricSeries := range seed.Metrics {
		server, ok := serverByID[serverID]
		if !ok {
			continue
		}
		for _, series := range metricSeries {
			for _, point := range series.Points {
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO metric_points (id, tenant_id, server_id, service_group_id, container_id, metric_key, unit, recorded_at, value)
					VALUES ($1, $2, $3, NULL, NULL, $4, $5, $6, $7)
				`, mustNewUUIDString(), server.TenantID, server.ID, series.Key, series.Unit, point.Timestamp, point.Value); err != nil {
					return err
				}
			}
		}
	}

	for serverID, bundle := range seed.ContainerMetrics {
		server, ok := serverByID[serverID]
		if !ok {
			continue
		}
		if err := s.insertSeedContainerMetricBundle(ctx, tx, server.TenantID, server.ID, bundle, containerIDToServiceID); err != nil {
			return err
		}
	}

	for serviceID, lines := range seed.Logs {
		for _, line := range lines {
			server, ok := serverByID[line.ServerID]
			if !ok {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO log_lines (
					id, tenant_id, server_id, service_group_id, container_id, level, message, occurred_at, container_name, service_tag, created_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
			`, line.ID, server.TenantID, line.ServerID, serviceID, line.ContainerID, line.Level, line.Message, line.Timestamp, line.ContainerName, line.ServiceTag); err != nil {
				return err
			}
		}
	}

	for _, agent := range seed.Agents {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO agents (
				id, tenant_id, server_id, name, api_key_hash, version, server_name, hostname, description, last_seen_at, enrolled_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO NOTHING
		`, agent.ID, agent.TenantID, agent.ServerID, agent.Name, hashSecret(agent.APIKey), agent.Version, agent.ServerName, agent.Hostname, agent.Description, agent.LastSeenAt, agent.EnrolledAt); err != nil {
			return err
		}
	}

	for _, event := range seedEventsFromServices(seed.Services) {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_logs (
				id, tenant_id, server_id, service_group_id, container_id, event_type, message, entity_name, occurred_at, created_at
			)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, NOW())
		`, event.ID, event.TenantID, event.ServerID, event.ServiceID, event.ContainerID, event.Type, event.Message, event.EntityName, event.Timestamp); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) BootstrapAdmin(tenantName, name, email, password string) (domain.User, error) {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()

	var userCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return domain.User{}, err
	}
	if userCount > 0 {
		return domain.User{}, ErrConflict
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}

	tenantID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}
	userID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}
	membershipID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}

	if tenantName == "" {
		tenantName = "Bifrost"
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tenants (id, name, created_at)
		VALUES ($1, $2, NOW())
	`, tenantID, tenantName); err != nil {
		return domain.User{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, tenant_id, email, password_hash, name, role, created_at, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), TRUE, NOW())
	`, userID, tenantID, strings.ToLower(strings.TrimSpace(email)), string(hashedPassword), name, string(domain.RoleAdmin)); err != nil {
		return domain.User{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tenant_memberships (id, tenant_id, user_id, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())
	`, membershipID, tenantID, userID, string(domain.RoleAdmin)); err != nil {
		return domain.User{}, err
	}

	token, err := s.createSession(ctx, tx, tenantID, userID)
	if err != nil {
		return domain.User{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:        userID,
		TenantID:  tenantID,
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Name:      name,
		Role:      domain.RoleAdmin,
		AuthToken: token,
	}, nil
}

func (s *PostgresStore) Authenticate(email, password string) (domain.User, error) {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()

	var (
		user     domain.User
		hash     string
		role     string
		tenantID string
	)
	err = tx.QueryRowContext(ctx, `
		SELECT u.id, tm.tenant_id, u.email, u.name, tm.role, u.password_hash
		FROM users u
		JOIN tenant_memberships tm ON tm.user_id = u.id
		WHERE lower(u.email) = lower($1)
		  AND u.is_active = TRUE
		  AND tm.is_active = TRUE
		ORDER BY tm.created_at ASC
		LIMIT 1
	`, email).Scan(&user.ID, &tenantID, &user.Email, &user.Name, &role, &hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return domain.User{}, ErrNotFound
	}

	token, err := s.createSession(ctx, tx, tenantID, user.ID)
	if err != nil {
		return domain.User{}, err
	}

	user.TenantID = tenantID
	user.Role = domain.UserRole(role)
	user.AuthToken = token

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, user.ID); err != nil {
		return domain.User{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (s *PostgresStore) UserByToken(token string) (domain.User, error) {
	ctx := context.Background()

	var (
		user     domain.User
		role     string
		tenantID string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, tm.tenant_id, u.email, u.name, tm.role
		FROM user_sessions us
		JOIN users u ON u.id = us.user_id
		JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.tenant_id = us.tenant_id
		WHERE us.session_token_hash = $1
		  AND us.revoked_at IS NULL
		  AND us.expires_at > NOW()
		  AND u.is_active = TRUE
		  AND tm.is_active = TRUE
		LIMIT 1
	`, hashSecret(token)).Scan(&user.ID, &tenantID, &user.Email, &user.Name, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}

	user.TenantID = tenantID
	user.Role = domain.UserRole(role)
	user.AuthToken = token

	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET last_seen_at = NOW()
		WHERE session_token_hash = $1
	`, hashSecret(token))

	return user, nil
}

func (s *PostgresStore) RevokeSession(token string) error {
	ctx := context.Background()

	result, err := s.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE session_token_hash = $1
		  AND revoked_at IS NULL
	`, hashSecret(token))
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresStore) UpdateUserName(userID, name string) (domain.User, error) {
	ctx := context.Background()
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.User{}, ErrConflict
	}

	var (
		user     domain.User
		role     string
		tenantID string
	)
	err := s.db.QueryRowContext(ctx, `
		UPDATE users u
		SET name = $2, updated_at = NOW()
		FROM tenant_memberships tm
		WHERE u.id = $1
		  AND tm.user_id = u.id
		  AND tm.is_active = TRUE
		RETURNING u.id, tm.tenant_id, u.email, u.name, tm.role
	`, userID, name).Scan(&user.ID, &tenantID, &user.Email, &user.Name, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}

	user.TenantID = tenantID
	user.Role = domain.UserRole(role)
	return user, nil
}

func (s *PostgresStore) ChangeUserPassword(userID, currentPassword, newPassword string) error {
	ctx := context.Background()

	var hash string
	if err := s.db.QueryRowContext(ctx, `
		SELECT password_hash
		FROM users
		WHERE id = $1
		  AND is_active = TRUE
	`, userID).Scan(&hash); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	nextHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
		  AND is_active = TRUE
	`, userID, string(nextHash))
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresStore) TenantSummary(tenantID string) (domain.TenantSummary, error) {
	ctx := context.Background()

	var summary domain.TenantSummary
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, name
		FROM tenants
		WHERE id = $1
	`, tenantID).Scan(&summary.TenantID, &summary.TenantName); err != nil {
		if err == sql.ErrNoRows {
			return domain.TenantSummary{}, ErrNotFound
		}
		return domain.TenantSummary{}, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT role, COUNT(*)
		FROM tenant_memberships
		WHERE tenant_id = $1
		  AND is_active = TRUE
		GROUP BY role
	`, tenantID)
	if err != nil {
		return domain.TenantSummary{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			role  string
			count int
		)
		if err := rows.Scan(&role, &count); err != nil {
			return domain.TenantSummary{}, err
		}
		switch role {
		case string(domain.RoleViewer):
			summary.ViewerCount = count
		default:
			summary.AdminCount += count
		}
	}
	if err := rows.Err(); err != nil {
		return domain.TenantSummary{}, err
	}

	return summary, nil
}

func (s *PostgresStore) ViewerAccess(tenantID string) (domain.ViewerAccess, error) {
	ctx := context.Background()

	access := domain.ViewerAccess{
		Viewers: []domain.ViewerAccount{},
		Invites: []domain.ViewerInvite{},
	}

	viewerRows, err := s.db.QueryContext(ctx, `
		SELECT u.id, tm.tenant_id, u.email, u.name, tm.role, tm.is_active, tm.disabled_at
		FROM tenant_memberships tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.tenant_id = $1
		  AND tm.role = 'viewer'
		ORDER BY lower(u.email) ASC
	`, tenantID)
	if err != nil {
		return domain.ViewerAccess{}, err
	}
	defer viewerRows.Close()

	for viewerRows.Next() {
		var (
			account    domain.ViewerAccount
			role       string
			isActive   bool
			disabledAt sql.NullTime
		)
		if err := viewerRows.Scan(&account.ID, &account.TenantID, &account.Email, &account.Name, &role, &isActive, &disabledAt); err != nil {
			return domain.ViewerAccess{}, err
		}
		account.Role = domain.UserRole(role)
		account.Status = "active"
		if !isActive {
			account.Status = "disabled"
		}
		if disabledAt.Valid {
			disabled := disabledAt.Time
			account.DisabledAt = &disabled
		}
		access.Viewers = append(access.Viewers, account)
	}
	if err := viewerRows.Err(); err != nil {
		return domain.ViewerAccess{}, err
	}

	inviteRows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, email, role, COALESCE(invited_by_user_id, ''), expires_at, created_at, accepted_at, revoked_at
		FROM user_invites
		WHERE tenant_id = $1
		  AND accepted_at IS NULL
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return domain.ViewerAccess{}, err
	}
	defer inviteRows.Close()

	for inviteRows.Next() {
		var (
			invite     domain.ViewerInvite
			role       string
			acceptedAt sql.NullTime
			revokedAt  sql.NullTime
		)
		if err := inviteRows.Scan(&invite.ID, &invite.TenantID, &invite.Email, &role, &invite.InvitedByUserID, &invite.ExpiresAt, &invite.CreatedAt, &acceptedAt, &revokedAt); err != nil {
			return domain.ViewerAccess{}, err
		}
		invite.Role = domain.UserRole(role)
		if acceptedAt.Valid {
			accepted := acceptedAt.Time
			invite.AcceptedAt = &accepted
		}
		if revokedAt.Valid {
			revoked := revokedAt.Time
			invite.RevokedAt = &revoked
		}
		invite.Status = inviteLifecycleStatus(invite.AcceptedAt, invite.RevokedAt, invite.ExpiresAt)
		access.Invites = append(access.Invites, invite)
	}
	if err := inviteRows.Err(); err != nil {
		return domain.ViewerAccess{}, err
	}

	return access, nil
}

func (s *PostgresStore) CreateViewerInvite(tenantID, invitedByUserID, email string) (domain.ViewerInvite, error) {
	ctx := context.Background()
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return domain.ViewerInvite{}, ErrConflict
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ViewerInvite{}, err
	}
	defer tx.Rollback()

	var existingUserCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users u
		JOIN tenant_memberships tm ON tm.user_id = u.id
		WHERE tm.tenant_id = $1
		  AND lower(u.email) = lower($2)
	`, tenantID, email).Scan(&existingUserCount); err != nil {
		return domain.ViewerInvite{}, err
	}
	if existingUserCount > 0 {
		return domain.ViewerInvite{}, ErrConflict
	}

	var existingInviteCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_invites
		WHERE tenant_id = $1
		  AND lower(email) = lower($2)
		  AND accepted_at IS NULL
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, tenantID, email).Scan(&existingInviteCount); err != nil {
		return domain.ViewerInvite{}, err
	}
	if existingInviteCount > 0 {
		return domain.ViewerInvite{}, ErrConflict
	}

	inviteID, err := newUUIDString()
	if err != nil {
		return domain.ViewerInvite{}, err
	}
	token, tokenHash, err := newOpaqueToken()
	if err != nil {
		return domain.ViewerInvite{}, err
	}

	now := time.Now().UTC()
	invite := domain.ViewerInvite{
		ID:              inviteID,
		TenantID:        tenantID,
		Email:           email,
		Role:            domain.RoleViewer,
		InvitedByUserID: invitedByUserID,
		ExpiresAt:       now.Add(7 * 24 * time.Hour),
		CreatedAt:       now,
		InviteToken:     token,
		Status:          "pending",
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_invites (
			id, tenant_id, email, role, invite_token_hash, invited_by_user_id, expires_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, invite.ID, invite.TenantID, invite.Email, string(invite.Role), tokenHash, invite.InvitedByUserID, invite.ExpiresAt, invite.CreatedAt); err != nil {
		return domain.ViewerInvite{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.ViewerInvite{}, err
	}

	return invite, nil
}

func (s *PostgresStore) CreateSystemOnboarding(tenantID, createdByUserID, name, description string) (domain.SystemOnboarding, error) {
	ctx := context.Background()
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
	agentID, err := newUUIDString()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	apiKey, apiKeyHash, err := newOpaqueToken()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}

	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO agents (
			id, tenant_id, server_id, name, api_key_hash, version, server_name, hostname, description, last_seen_at, enrolled_at
		)
		VALUES ($1, $2, NULL, $3, $4, $5, $6, '', $7, NULL, $8)
	`, agentID, tenantID, name, apiKeyHash, "pending", name, description, now); err != nil {
		return domain.SystemOnboarding{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO system_onboardings (
			id, tenant_id, server_id, agent_id, name, description, status, created_by_user_id, created_at, connected_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'awaiting_connection', $7, $8, NULL)
	`, onboardingID, tenantID, serverID, agentID, name, description, createdByUserID, now); err != nil {
		return domain.SystemOnboarding{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.SystemOnboarding{}, err
	}

	return domain.SystemOnboarding{
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
	}, nil
}

func (s *PostgresStore) ListSystemOnboardings(tenantID string) ([]domain.SystemOnboarding, error) {
	ctx := context.Background()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, server_id, agent_id, name, description, status, COALESCE(created_by_user_id, ''), created_at, connected_at
		FROM system_onboardings
		WHERE tenant_id = $1
		ORDER BY
			CASE status
				WHEN 'awaiting_connection' THEN 0
				WHEN 'connected' THEN 1
				ELSE 2
			END,
			COALESCE(connected_at, created_at) DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.SystemOnboarding{}
	for rows.Next() {
		var connectedAt sql.NullTime
		onboarding := domain.SystemOnboarding{}
		if err := rows.Scan(
			&onboarding.ID,
			&onboarding.TenantID,
			&onboarding.ServerID,
			&onboarding.AgentID,
			&onboarding.Name,
			&onboarding.Description,
			&onboarding.Status,
			&onboarding.CreatedByUserID,
			&onboarding.CreatedAt,
			&connectedAt,
		); err != nil {
			return nil, err
		}
		if connectedAt.Valid {
			onboarding.ConnectedAt = &connectedAt.Time
		}
		result = append(result, onboarding)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *PostgresStore) SystemOnboardingByID(tenantID, onboardingID string) (domain.SystemOnboarding, error) {
	ctx := context.Background()

	var connectedAt sql.NullTime
	onboarding := domain.SystemOnboarding{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, server_id, agent_id, name, description, status, COALESCE(created_by_user_id, ''), created_at, connected_at
		FROM system_onboardings
		WHERE tenant_id = $1 AND id = $2
		LIMIT 1
	`, tenantID, onboardingID).Scan(
		&onboarding.ID,
		&onboarding.TenantID,
		&onboarding.ServerID,
		&onboarding.AgentID,
		&onboarding.Name,
		&onboarding.Description,
		&onboarding.Status,
		&onboarding.CreatedByUserID,
		&onboarding.CreatedAt,
		&connectedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.SystemOnboarding{}, ErrNotFound
		}
		return domain.SystemOnboarding{}, err
	}
	if connectedAt.Valid {
		onboarding.ConnectedAt = &connectedAt.Time
	}
	return onboarding, nil
}

func (s *PostgresStore) CancelSystemOnboarding(tenantID, onboardingID string) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var agentID string
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT agent_id, status
		FROM system_onboardings
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, onboardingID).Scan(&agentID, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	if status != "awaiting_connection" {
		return ErrConflict
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM system_onboardings
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, onboardingID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM agents
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, agentID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) ReissueSystemOnboardingCredentials(tenantID, onboardingID string) (domain.SystemOnboarding, error) {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	defer tx.Rollback()

	var connectedAt sql.NullTime
	onboarding := domain.SystemOnboarding{}
	err = tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, server_id, agent_id, name, description, status, COALESCE(created_by_user_id, ''), created_at, connected_at
		FROM system_onboardings
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, onboardingID).Scan(
		&onboarding.ID,
		&onboarding.TenantID,
		&onboarding.ServerID,
		&onboarding.AgentID,
		&onboarding.Name,
		&onboarding.Description,
		&onboarding.Status,
		&onboarding.CreatedByUserID,
		&onboarding.CreatedAt,
		&connectedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.SystemOnboarding{}, ErrNotFound
		}
		return domain.SystemOnboarding{}, err
	}
	if onboarding.Status != "awaiting_connection" {
		return domain.SystemOnboarding{}, ErrConflict
	}
	if connectedAt.Valid {
		onboarding.ConnectedAt = &connectedAt.Time
	}

	apiKey, apiKeyHash, err := newOpaqueToken()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE agents
		SET api_key_hash = $3
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, onboarding.AgentID, apiKeyHash)
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return domain.SystemOnboarding{}, err
	}
	if rows == 0 {
		return domain.SystemOnboarding{}, ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return domain.SystemOnboarding{}, err
	}

	onboarding.APIKey = apiKey
	return onboarding, nil
}

func (s *PostgresStore) SelfEnrollPendingAgent(agentID, serverID string) (domain.Agent, error) {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Agent{}, err
	}
	defer tx.Rollback()

	var (
		agent            domain.Agent
		storedServerID   sql.NullString
		onboardingID     sql.NullString
		onboardingStatus sql.NullString
	)
	err = tx.QueryRowContext(ctx, `
		SELECT a.id, a.tenant_id, a.name, a.version, COALESCE(a.last_seen_at, a.enrolled_at), a.enrolled_at,
		       a.server_name, a.hostname, a.description, a.server_id,
		       o.id, o.status
		FROM agents a
		LEFT JOIN system_onboardings o ON o.agent_id = a.id
		WHERE a.id = $1
		FOR UPDATE OF a
	`, agentID).Scan(
		&agent.ID,
		&agent.TenantID,
		&agent.Name,
		&agent.Version,
		&agent.LastSeenAt,
		&agent.EnrolledAt,
		&agent.ServerName,
		&agent.Hostname,
		&agent.Description,
		&storedServerID,
		&onboardingID,
		&onboardingStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Agent{}, ErrNotFound
		}
		return domain.Agent{}, err
	}
	if storedServerID.Valid {
		agent.ServerID = storedServerID.String
	}
	if agent.Version != "pending" {
		return domain.Agent{}, ErrConflict
	}
	if !onboardingID.Valid || !onboardingStatus.Valid || onboardingStatus.String != "awaiting_connection" {
		return domain.Agent{}, ErrConflict
	}

	var onboardingServerID string
	err = tx.QueryRowContext(ctx, `
		SELECT server_id
		FROM system_onboardings
		WHERE id = $1 AND agent_id = $2
		FOR UPDATE
	`, onboardingID.String, agentID).Scan(&onboardingServerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Agent{}, ErrNotFound
		}
		return domain.Agent{}, err
	}
	if onboardingServerID != serverID {
		return domain.Agent{}, ErrConflict
	}

	newAPIKey, newAPIKeyHash, err := newOpaqueToken()
	if err != nil {
		return domain.Agent{}, err
	}

	// The real servers row is created on first snapshot ingest, so keep the
	// pending agent detached from server_id during enrollment to avoid FK races.
	if _, err := tx.ExecContext(ctx, `
		UPDATE agents
		SET api_key_hash = $2,
		    version = 'enrolled',
		    last_seen_at = NOW()
		WHERE id = $1
	`, agentID, newAPIKeyHash); err != nil {
		return domain.Agent{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Agent{}, err
	}

	agent.ServerID = onboardingServerID
	agent.Version = "enrolled"
	agent.APIKey = newAPIKey
	agent.LastSeenAt = time.Now().UTC()
	return agent, nil
}

func (s *PostgresStore) InviteByToken(token string) (domain.ViewerInvite, error) {
	ctx := context.Background()

	var (
		invite     domain.ViewerInvite
		role       string
		acceptedAt sql.NullTime
		revokedAt  sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, email, role, COALESCE(invited_by_user_id, ''), expires_at, created_at, accepted_at, revoked_at
		FROM user_invites
		WHERE invite_token_hash = $1
		LIMIT 1
	`, hashSecret(token)).Scan(&invite.ID, &invite.TenantID, &invite.Email, &role, &invite.InvitedByUserID, &invite.ExpiresAt, &invite.CreatedAt, &acceptedAt, &revokedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ViewerInvite{}, ErrNotFound
		}
		return domain.ViewerInvite{}, err
	}

	invite.Role = domain.UserRole(role)
	if acceptedAt.Valid {
		accepted := acceptedAt.Time
		invite.AcceptedAt = &accepted
	}
	if revokedAt.Valid {
		revoked := revokedAt.Time
		invite.RevokedAt = &revoked
	}
	invite.Status = inviteLifecycleStatus(invite.AcceptedAt, invite.RevokedAt, invite.ExpiresAt)

	if invite.Status != "pending" {
		return domain.ViewerInvite{}, ErrConflict
	}

	return invite, nil
}

func (s *PostgresStore) AcceptViewerInvite(token, name, password string) (domain.User, error) {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()

	var (
		invite     domain.ViewerInvite
		role       string
		acceptedAt sql.NullTime
		revokedAt  sql.NullTime
	)
	err = tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, email, role, COALESCE(invited_by_user_id, ''), expires_at, created_at, accepted_at, revoked_at
		FROM user_invites
		WHERE invite_token_hash = $1
		FOR UPDATE
	`, hashSecret(token)).Scan(&invite.ID, &invite.TenantID, &invite.Email, &role, &invite.InvitedByUserID, &invite.ExpiresAt, &invite.CreatedAt, &acceptedAt, &revokedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}

	invite.Role = domain.UserRole(role)
	if acceptedAt.Valid {
		accepted := acceptedAt.Time
		invite.AcceptedAt = &accepted
	}
	if revokedAt.Valid {
		revoked := revokedAt.Time
		invite.RevokedAt = &revoked
	}
	if inviteLifecycleStatus(invite.AcceptedAt, invite.RevokedAt, invite.ExpiresAt) != "pending" {
		return domain.User{}, ErrConflict
	}

	var existingUserCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users u
		JOIN tenant_memberships tm ON tm.user_id = u.id
		WHERE tm.tenant_id = $1
		  AND lower(u.email) = lower($2)
	`, invite.TenantID, invite.Email).Scan(&existingUserCount); err != nil {
		return domain.User{}, err
	}
	if existingUserCount > 0 {
		return domain.User{}, ErrConflict
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}

	userID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}
	membershipID, err := newUUIDString()
	if err != nil {
		return domain.User{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, tenant_id, email, password_hash, name, role, created_at, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), TRUE, NOW())
	`, userID, invite.TenantID, invite.Email, string(hashedPassword), name, string(domain.RoleViewer)); err != nil {
		return domain.User{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tenant_memberships (id, tenant_id, user_id, role, is_active, invited_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, TRUE, NULLIF($5, ''), NOW(), NOW())
	`, membershipID, invite.TenantID, userID, string(domain.RoleViewer), invite.InvitedByUserID); err != nil {
		return domain.User{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE user_invites
		SET accepted_by_user_id = $2,
		    accepted_at = NOW()
		WHERE id = $1
	`, invite.ID, userID); err != nil {
		return domain.User{}, err
	}

	sessionToken, err := s.createSession(ctx, tx, invite.TenantID, userID)
	if err != nil {
		return domain.User{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:        userID,
		TenantID:  invite.TenantID,
		Email:     invite.Email,
		Name:      name,
		Role:      domain.RoleViewer,
		AuthToken: sessionToken,
	}, nil
}

func (s *PostgresStore) RevokeViewerInvite(tenantID, inviteID string) error {
	ctx := context.Background()

	result, err := s.db.ExecContext(ctx, `
		UPDATE user_invites
		SET revoked_at = NOW()
		WHERE tenant_id = $1
		  AND id = $2
		  AND accepted_at IS NULL
		  AND revoked_at IS NULL
	`, tenantID, inviteID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DisableViewer(tenantID, viewerUserID string) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE tenant_memberships
		SET is_active = FALSE,
		    disabled_at = NOW(),
		    updated_at = NOW()
		WHERE tenant_id = $1
		  AND user_id = $2
		  AND role = 'viewer'
		  AND is_active = TRUE
	`, tenantID, viewerUserID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE tenant_id = $1
		  AND user_id = $2
		  AND revoked_at IS NULL
	`, tenantID, viewerUserID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) DeleteViewer(tenantID, viewerUserID string) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingUserID string
	err = tx.QueryRowContext(ctx, `
		SELECT u.id
		FROM users u
		JOIN tenant_memberships tm ON tm.user_id = u.id
		WHERE tm.tenant_id = $1
		  AND tm.user_id = $2
		  AND tm.role = 'viewer'
		LIMIT 1
	`, tenantID, viewerUserID).Scan(&existingUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE tenant_id = $1
		  AND user_id = $2
		  AND revoked_at IS NULL
	`, tenantID, viewerUserID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM tenant_memberships
		WHERE tenant_id = $1
		  AND user_id = $2
		  AND role = 'viewer'
	`, tenantID, viewerUserID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM users
		WHERE id = $1
		  AND NOT EXISTS (
		    SELECT 1
		    FROM tenant_memberships tm
		    WHERE tm.user_id = users.id
		  )
	`, viewerUserID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) ListServers(tenantID string) []domain.Server {
	ctx := context.Background()
	servers, err := s.queryServers(ctx, tenantID)
	if err != nil {
		return nil
	}
	return servers
}

func (s *PostgresStore) ServerByID(tenantID, serverID string) (domain.Server, error) {
	return s.queryServerByID(context.Background(), tenantID, serverID)
}

func (s *PostgresStore) ServicesByServer(tenantID, serverID string) []domain.Service {
	ctx := context.Background()
	services, err := s.queryServicesByServer(ctx, tenantID, serverID)
	if err != nil {
		return nil
	}
	return services
}

func (s *PostgresStore) ServiceByID(tenantID, serviceID string) (domain.Service, error) {
	return s.queryServiceByID(context.Background(), tenantID, serviceID)
}

func (s *PostgresStore) ProjectByID(tenantID, serverID, projectID string) (domain.Service, error) {
	project, err := s.queryServiceByID(context.Background(), tenantID, projectID)
	if err != nil {
		return domain.Service{}, err
	}
	if project.ServerID != serverID || project.ComposeProject == "" {
		return domain.Service{}, ErrNotFound
	}
	return project, nil
}

func (s *PostgresStore) ProjectsByServer(tenantID, serverID string) []domain.Service {
	services, err := s.queryServicesByServer(context.Background(), tenantID, serverID)
	if err != nil {
		return nil
	}
	projects := make([]domain.Service, 0, len(services))
	for _, service := range services {
		if service.ComposeProject != "" {
			projects = append(projects, service)
		}
	}
	return projects
}

func (s *PostgresStore) StandaloneContainersByServer(tenantID, serverID string) []domain.Container {
	services, err := s.queryServicesByServer(context.Background(), tenantID, serverID)
	if err != nil {
		return nil
	}
	containers := make([]domain.Container, 0)
	for _, service := range services {
		if service.ComposeProject != "" {
			continue
		}
		for _, container := range service.Containers {
			containerCopy := container
			containerCopy.Ports = append([]string(nil), container.Ports...)
			containers = append(containers, containerCopy)
		}
	}
	return containers
}

func (s *PostgresStore) MetricsByServer(serverID string) []domain.MetricSeries {
	series, err := s.queryServerMetrics(context.Background(), serverID)
	if err != nil {
		return nil
	}
	return series
}

func (s *PostgresStore) ContainerMetricsByServer(serverID string) domain.ContainerMetricBundle {
	bundle, err := s.queryServerContainerMetricBundle(context.Background(), serverID)
	if err != nil {
		return domain.ContainerMetricBundle{}
	}
	return bundle
}

func (s *PostgresStore) LogsByService(serviceID string) []domain.LogLine {
	lines, err := s.queryLogs(context.Background(), serviceID, "")
	if err != nil {
		return nil
	}
	return lines
}

func (s *PostgresStore) LogsByContainer(serviceID, containerID string) []domain.LogLine {
	lines, err := s.queryLogs(context.Background(), serviceID, containerID)
	if err != nil {
		return nil
	}
	return lines
}

func (s *PostgresStore) ServerBundle(tenantID, serverID string) (domain.ServerBundle, error) {
	server, err := s.queryServerByInternalID(context.Background(), tenantID, serverID)
	if err != nil {
		return domain.ServerBundle{}, err
	}

	services, err := s.queryServicesByServer(context.Background(), tenantID, serverID)
	if err != nil {
		return domain.ServerBundle{}, err
	}

	metrics, err := s.queryServerMetrics(context.Background(), serverID)
	if err != nil {
		return domain.ServerBundle{}, err
	}

	containerMetrics, err := s.queryServerContainerMetricBundle(context.Background(), serverID)
	if err != nil {
		return domain.ServerBundle{}, err
	}

	return domain.ServerBundle{
		Server:           server,
		Services:         services,
		Metrics:          metrics,
		ContainerMetrics: filterContainerMetricBundleByKeys(containerMetrics, currentContainerIDs(services)),
	}, nil
}

func (s *PostgresStore) ContainerByID(tenantID, serverID, containerID string) (domain.Container, domain.Service, error) {
	service, container, err := s.queryContainerWithService(context.Background(), tenantID, serverID, containerID)
	if err != nil {
		return domain.Container{}, domain.Service{}, err
	}
	return container, service, nil
}

func (s *PostgresStore) ProjectMetrics(tenantID, serverID, projectID string) (domain.ContainerMetricBundle, error) {
	project, err := s.ProjectByID(tenantID, serverID, projectID)
	if err != nil {
		return domain.ContainerMetricBundle{}, err
	}

	containerIDs := make([]string, 0, len(project.Containers))
	for _, container := range project.Containers {
		containerIDs = append(containerIDs, container.ID)
	}

	bundle, err := s.queryServerContainerMetricBundle(context.Background(), serverID)
	if err != nil {
		return domain.ContainerMetricBundle{}, err
	}
	return filterContainerMetricBundleByKeys(bundle, containerIDs), nil
}

func (s *PostgresStore) ContainerMetrics(tenantID, serverID, containerID string) (domain.ContainerMetricHistory, domain.Container, domain.Service, error) {
	service, container, err := s.queryContainerWithService(context.Background(), tenantID, serverID, containerID)
	if err != nil {
		return domain.ContainerMetricHistory{}, domain.Container{}, domain.Service{}, err
	}

	bundle, err := s.queryServerContainerMetricBundle(context.Background(), serverID)
	if err != nil {
		return domain.ContainerMetricHistory{}, domain.Container{}, domain.Service{}, err
	}
	return containerMetricHistoryByKey(bundle, container.ID), container, service, nil
}

func (s *PostgresStore) ProjectEvents(tenantID, serverID, projectID string) ([]domain.EventLog, domain.Service, error) {
	project, err := s.ProjectByID(tenantID, serverID, projectID)
	if err != nil {
		return nil, domain.Service{}, err
	}

	events, err := s.queryEvents(context.Background(), project.ID, "")
	if err != nil {
		return nil, domain.Service{}, err
	}
	return events, project, nil
}

func (s *PostgresStore) ContainerEvents(tenantID, serverID, containerID string) ([]domain.EventLog, domain.Container, domain.Service, error) {
	service, container, err := s.queryContainerWithService(context.Background(), tenantID, serverID, containerID)
	if err != nil {
		return nil, domain.Container{}, domain.Service{}, err
	}

	events, err := s.queryEvents(context.Background(), service.ID, container.ID)
	if err != nil {
		return nil, domain.Container{}, domain.Service{}, err
	}
	return events, container, service, nil
}

func (s *PostgresStore) ContainerEnv(tenantID, serverID, containerID string) (map[string]string, domain.Container, domain.Service, error) {
	service, container, err := s.queryContainerWithService(context.Background(), tenantID, serverID, containerID)
	if err != nil {
		return nil, domain.Container{}, domain.Service{}, err
	}
	return deriveContainerEnv(container, service), container, service, nil
}

func (s *PostgresStore) EnrollAgent(agent domain.Agent) (domain.Agent, error) {
	ctx := context.Background()
	now := time.Now().UTC()
	agent.EnrolledAt = now
	agent.LastSeenAt = now
	if agent.APIKey == "" {
		agent.APIKey = agent.ID + "-key"
	}

	var serverID any
	if agent.ServerID != "" {
		var existingServerID string
		err := s.db.QueryRowContext(ctx, `
			SELECT id
			FROM servers
			WHERE id = $1 AND tenant_id = $2
		`, agent.ServerID, agent.TenantID).Scan(&existingServerID)
		switch {
		case err == nil:
			serverID = existingServerID
		case err == sql.ErrNoRows:
			agent.ServerID = ""
			serverID = nil
		default:
			return domain.Agent{}, err
		}
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (
			id, tenant_id, server_id, name, api_key_hash, version, server_name, hostname, description, last_seen_at, enrolled_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE
		SET tenant_id = EXCLUDED.tenant_id,
		    server_id = EXCLUDED.server_id,
		    name = EXCLUDED.name,
		    api_key_hash = EXCLUDED.api_key_hash,
		    version = EXCLUDED.version,
		    server_name = EXCLUDED.server_name,
		    hostname = EXCLUDED.hostname,
		    description = EXCLUDED.description,
		    last_seen_at = EXCLUDED.last_seen_at
	`, agent.ID, agent.TenantID, serverID, agent.Name, hashSecret(agent.APIKey), agent.Version, agent.ServerName, agent.Hostname, agent.Description, agent.LastSeenAt, agent.EnrolledAt); err != nil {
		return domain.Agent{}, err
	}

	return agent, nil
}

func (s *PostgresStore) AgentByAPIKey(apiKey string) (domain.Agent, error) {
	ctx := context.Background()

	var agent domain.Agent
	err := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, COALESCE(server_id, ''), name, version, COALESCE(last_seen_at, enrolled_at), enrolled_at, server_name, hostname, description
		FROM agents
		WHERE api_key_hash = $1
		LIMIT 1
	`, hashSecret(apiKey)).Scan(
		&agent.ID,
		&agent.TenantID,
		&agent.ServerID,
		&agent.Name,
		&agent.Version,
		&agent.LastSeenAt,
		&agent.EnrolledAt,
		&agent.ServerName,
		&agent.Hostname,
		&agent.Description,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Agent{}, ErrNotFound
		}
		return domain.Agent{}, err
	}

	return agent, nil
}

func (s *PostgresStore) UpdateAgentLastSeen(agentID string) error {
	ctx := context.Background()

	result, err := s.db.ExecContext(ctx, `
		UPDATE agents
		SET last_seen_at = NOW()
		WHERE id = $1
	`, agentID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Ingest(payload domain.IngestPayload) error {
	ctx := context.Background()

	agent, err := s.agentByID(ctx, payload.AgentID)
	if err != nil {
		return err
	}

	serverID := payload.Server.ID
	if serverID == "" {
		serverID = agent.ServerID
	}
	if serverID == "" {
		return fmt.Errorf("missing server id")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	collectedAt := payload.Server.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO servers (
			id, tenant_id, name, hostname, public_ip, agent_version, status, last_seen_at, uptime_seconds,
			cpu_usage_pct, memory_usage_pct, disk_usage_pct, network_rx_mb, network_tx_mb, load_average,
			os, kernel, cpu_model, cpu_cores, cpu_threads, total_memory_gb, total_disk_gb, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, NOW(), NOW()
		)
		ON CONFLICT (id) DO UPDATE
		SET tenant_id = EXCLUDED.tenant_id,
		    name = EXCLUDED.name,
		    hostname = EXCLUDED.hostname,
		    public_ip = EXCLUDED.public_ip,
		    agent_version = EXCLUDED.agent_version,
		    status = EXCLUDED.status,
		    last_seen_at = EXCLUDED.last_seen_at,
		    uptime_seconds = EXCLUDED.uptime_seconds,
		    cpu_usage_pct = EXCLUDED.cpu_usage_pct,
		    memory_usage_pct = EXCLUDED.memory_usage_pct,
		    disk_usage_pct = EXCLUDED.disk_usage_pct,
		    network_rx_mb = EXCLUDED.network_rx_mb,
		    network_tx_mb = EXCLUDED.network_tx_mb,
		    load_average = EXCLUDED.load_average,
		    os = EXCLUDED.os,
		    kernel = EXCLUDED.kernel,
		    cpu_model = EXCLUDED.cpu_model,
		    cpu_cores = EXCLUDED.cpu_cores,
		    cpu_threads = EXCLUDED.cpu_threads,
		    total_memory_gb = EXCLUDED.total_memory_gb,
		    total_disk_gb = EXCLUDED.total_disk_gb,
		    updated_at = NOW()
	`, serverID, agent.TenantID, payload.Server.Name, payload.Server.Hostname, payload.Server.PublicIP, payload.Server.AgentVersion,
		normalizeServerStatus(payload.Server.Status), collectedAt, payload.Server.UptimeSeconds, payload.Server.CPUUsagePct,
		payload.Server.MemoryUsagePct, payload.Server.DiskUsagePct, payload.Server.NetworkRXMB, payload.Server.NetworkTXMB, payload.Server.LoadAverage,
		payload.Server.OS, payload.Server.Kernel, payload.Server.CPUModel, payload.Server.CPUCores, payload.Server.CPUThreads,
		payload.Server.TotalMemoryGB, payload.Server.TotalDiskGB); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE agents
		SET server_id = $2, last_seen_at = NOW(), server_name = $3, hostname = $4
		WHERE id = $1
	`, agent.ID, serverID, payload.Server.Name, payload.Server.Hostname); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE system_onboardings
		SET status = 'connected',
		    connected_at = COALESCE(connected_at, $2)
		WHERE agent_id = $1
	`, agent.ID, collectedAt); err != nil {
		return err
	}

	existingServices, err := s.queryServicesByServer(ctx, agent.TenantID, serverID)
	if err != nil {
		return err
	}
	existingServicesByRuntimeKey := make(map[string]domain.Service, len(existingServices))
	existingContainersByServiceRuntimeKey := make(map[string]map[string]domain.Container, len(existingServices))
	for _, existingService := range existingServices {
		runtimeKey := monitoringServiceRuntimeKey(existingService.ComposeProject, existingService.Name)
		existingServicesByRuntimeKey[runtimeKey] = existingService
		containersByRuntimeKey := make(map[string]domain.Container, len(existingService.Containers))
		for _, existingContainer := range existingService.Containers {
			containersByRuntimeKey[monitoringContainerRuntimeKey(existingContainer.Name)] = existingContainer
		}
		existingContainersByServiceRuntimeKey[runtimeKey] = containersByRuntimeKey
	}

	incomingServiceIDs := make([]string, 0, len(payload.Server.Services))
	serviceMaxLogTime := map[string]time.Time{}
	serviceIDMap := make(map[string]string, len(payload.Server.Services))
	containerIDMap := map[string]string{}
	canonicalServices := make([]domain.ServiceSnapshot, 0, len(payload.Server.Services))

	for _, serviceSnapshot := range payload.Server.Services {
		runtimeServiceKey := monitoringServiceRuntimeKey(serviceSnapshot.ComposeProject, serviceSnapshot.Name)
		serviceID := resolveCanonicalMonitoringID(serviceSnapshot.ID)
		if existingService, ok := existingServicesByRuntimeKey[runtimeServiceKey]; ok {
			serviceID = existingService.ID
		}
		serviceIDMap[serviceSnapshot.ID] = serviceID
		incomingServiceIDs = append(incomingServiceIDs, serviceID)
		previousContainersByID := make(map[string]domain.Container)
		if existingService, ok := existingServicesByRuntimeKey[runtimeServiceKey]; ok {
			for _, previous := range existingService.Containers {
				previousContainersByID[previous.ID] = previous
			}
		}
		previousContainersByRuntimeKey := existingContainersByServiceRuntimeKey[runtimeServiceKey]

		canonicalServiceSnapshot := serviceSnapshot
		canonicalServiceSnapshot.ID = serviceID
		canonicalServiceSnapshot.Containers = make([]domain.ContainerSnapshot, 0, len(serviceSnapshot.Containers))

		serviceStatus := normalizeServiceStatus(serviceSnapshot.Status)
		restartCount := 0
		for _, containerSnapshot := range serviceSnapshot.Containers {
			restartCount += containerSnapshot.RestartCount
			serviceStatus = rollupServiceStatus(serviceStatus, normalizeContainerStatus(containerSnapshot.Status), normalizeHealth(containerSnapshot.Health))
		}

		ports, err := marshalStrings(serviceSnapshot.PublishedPorts)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO service_groups (
				id, tenant_id, server_id, name, compose_project, status, container_count, restart_count,
				published_ports, updated_at, last_log_timestamp, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NULL, NOW())
			ON CONFLICT (id) DO UPDATE
			SET tenant_id = EXCLUDED.tenant_id,
			    server_id = EXCLUDED.server_id,
			    name = EXCLUDED.name,
			    compose_project = EXCLUDED.compose_project,
			    status = EXCLUDED.status,
			    container_count = EXCLUDED.container_count,
			    restart_count = EXCLUDED.restart_count,
			    published_ports = EXCLUDED.published_ports,
			    updated_at = NOW()
		`, serviceID, agent.TenantID, serverID, serviceSnapshot.Name, serviceSnapshot.ComposeProject, serviceStatus, len(serviceSnapshot.Containers), restartCount, ports); err != nil {
			return err
		}

		incomingContainerIDs := make([]string, 0, len(serviceSnapshot.Containers))
		service := domain.Service{
			ID:             serviceID,
			TenantID:       agent.TenantID,
			ServerID:       serverID,
			Name:           serviceSnapshot.Name,
			ComposeProject: serviceSnapshot.ComposeProject,
			Status:         serviceStatus,
			ContainerCount: len(serviceSnapshot.Containers),
			RestartCount:   restartCount,
			PublishedPorts: append([]string(nil), serviceSnapshot.PublishedPorts...),
			Containers:     []domain.Container{},
		}
		for _, containerSnapshot := range serviceSnapshot.Containers {
			containerID := resolveCanonicalMonitoringID(containerSnapshot.ID)
			if previousContainer, ok := previousContainersByRuntimeKey[monitoringContainerRuntimeKey(containerSnapshot.Name)]; ok {
				containerID = previousContainer.ID
			}
			containerIDMap[containerSnapshot.ID] = containerID
			incomingContainerIDs = append(incomingContainerIDs, containerID)

			canonicalContainerSnapshot := containerSnapshot
			canonicalContainerSnapshot.ID = containerID
			canonicalServiceSnapshot.Containers = append(canonicalServiceSnapshot.Containers, canonicalContainerSnapshot)

			containerPorts, err := marshalStrings(containerSnapshot.Ports)
			if err != nil {
				return err
			}

			lastSeenAt := containerSnapshot.LastSeenAt
			if lastSeenAt.IsZero() {
				lastSeenAt = collectedAt
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO containers (
					id, service_group_id, name, image, status, health, cpu_usage_pct, memory_mb, network_mb,
					restart_count, uptime, ports, command, last_seen_at, created_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
				ON CONFLICT (id) DO UPDATE
				SET service_group_id = EXCLUDED.service_group_id,
				    name = EXCLUDED.name,
				    image = EXCLUDED.image,
				    status = EXCLUDED.status,
				    health = EXCLUDED.health,
				    cpu_usage_pct = EXCLUDED.cpu_usage_pct,
				    memory_mb = EXCLUDED.memory_mb,
				    network_mb = EXCLUDED.network_mb,
				    restart_count = EXCLUDED.restart_count,
				    uptime = EXCLUDED.uptime,
				    ports = EXCLUDED.ports,
				    command = EXCLUDED.command,
				    last_seen_at = EXCLUDED.last_seen_at
			`, containerID, serviceID, containerSnapshot.Name, containerSnapshot.Image,
				normalizeContainerStatus(containerSnapshot.Status), normalizeHealth(containerSnapshot.Health),
				containerSnapshot.CPUUsagePct, containerSnapshot.MemoryMB, containerSnapshot.NetworkMB,
				containerSnapshot.RestartCount, containerSnapshot.Uptime, containerPorts, containerSnapshot.Command, lastSeenAt); err != nil {
				return err
			}

			if err := s.insertContainerMetric(ctx, tx, agent.TenantID, serverID, serviceID, containerID, "cpu_usage_pct", "%", collectedAt, containerSnapshot.CPUUsagePct); err != nil {
				return err
			}
			if err := s.insertContainerMetric(ctx, tx, agent.TenantID, serverID, serviceID, containerID, "memory_mb", "MB", collectedAt, containerSnapshot.MemoryMB); err != nil {
				return err
			}
			if err := s.insertContainerMetric(ctx, tx, agent.TenantID, serverID, serviceID, containerID, "network_mb", "MB/s", collectedAt, containerSnapshot.NetworkMB); err != nil {
				return err
			}

			currentContainer := domain.Container{
				ID:           containerID,
				ServiceID:    serviceID,
				Name:         containerSnapshot.Name,
				Image:        containerSnapshot.Image,
				Status:       normalizeContainerStatus(containerSnapshot.Status),
				Health:       normalizeHealth(containerSnapshot.Health),
				CPUUsagePct:  containerSnapshot.CPUUsagePct,
				MemoryMB:     containerSnapshot.MemoryMB,
				NetworkMB:    containerSnapshot.NetworkMB,
				RestartCount: containerSnapshot.RestartCount,
				Uptime:       containerSnapshot.Uptime,
				Ports:        append([]string(nil), containerSnapshot.Ports...),
				Command:      containerSnapshot.Command,
				LastSeenAt:   lastSeenAt,
			}
			service.Containers = append(service.Containers, currentContainer)

			var previous *domain.Container
			if prior, ok := previousContainersByID[currentContainer.ID]; ok {
				priorCopy := prior
				previous = &priorCopy
			}
			for _, event := range buildContainerStateEvents(agent.TenantID, serverID, service, previous, &currentContainer, collectedAt) {
				if err := s.insertEvent(ctx, tx, event); err != nil {
					return err
				}
			}
			delete(previousContainersByID, currentContainer.ID)
		}

		canonicalServices = append(canonicalServices, canonicalServiceSnapshot)

		for _, previous := range previousContainersByID {
			previousCopy := previous
			for _, event := range buildContainerStateEvents(agent.TenantID, serverID, service, &previousCopy, nil, collectedAt) {
				if err := s.insertEvent(ctx, tx, event); err != nil {
					return err
				}
			}
		}

		if err := s.deleteStaleContainers(ctx, tx, serviceID, incomingContainerIDs); err != nil {
			return err
		}
	}

	if err := s.deleteStaleServices(ctx, tx, serverID, incomingServiceIDs); err != nil {
		return err
	}

	for _, metric := range payload.Metrics {
		serviceID := metric.ServiceID
		if mappedServiceID, ok := serviceIDMap[metric.ServiceID]; ok {
			serviceID = mappedServiceID
		}
		for _, point := range metric.Points {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO metric_points (id, tenant_id, server_id, service_group_id, container_id, metric_key, unit, recorded_at, value)
				VALUES ($1, $2, $3, NULLIF($4, ''), NULL, $5, $6, $7, $8)
			`, mustNewUUIDString(), agent.TenantID, serverID, serviceID, metric.Key, metric.Unit, point.Timestamp, point.Value); err != nil {
				return err
			}
		}
	}

	for _, logLine := range payload.Logs {
		serviceID := serviceIDMap[logLine.ServiceID]
		containerID := containerIDMap[logLine.ContainerID]
		containerName, serviceTag := lookupLogContext(canonicalServices, serviceID, containerID)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO log_lines (
				id, tenant_id, server_id, service_group_id, container_id, level, message, occurred_at, container_name, service_tag, created_at
			)
			VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, $9, $10, NOW())
		`, mustNewUUIDString(), agent.TenantID, serverID, serviceID, containerID, logLine.Level, logLine.Message, logLine.Timestamp, containerName, serviceTag); err != nil {
			return err
		}

		if current := serviceMaxLogTime[serviceID]; logLine.Timestamp.After(current) {
			serviceMaxLogTime[serviceID] = logLine.Timestamp
		}
	}

	for serviceID, timestamp := range serviceMaxLogTime {
		if _, err := tx.ExecContext(ctx, `
			UPDATE service_groups
			SET last_log_timestamp = $2, updated_at = NOW()
			WHERE id = $1
		`, serviceID, timestamp); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) queryServers(ctx context.Context, tenantID string) ([]domain.Server, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, name, hostname, public_ip, agent_version, status, last_seen_at, uptime_seconds,
		       cpu_usage_pct, memory_usage_pct, disk_usage_pct, network_rx_mb, network_tx_mb, load_average,
		       os, kernel, cpu_model, cpu_cores, cpu_threads, total_memory_gb, total_disk_gb
		FROM servers
		WHERE tenant_id = $1
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]domain.Server, 0)
	for rows.Next() {
		server, scanErr := scanServer(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		servers = append(servers, server)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return servers, nil
}

func (s *PostgresStore) queryServerByInternalID(ctx context.Context, tenantID, serverID string) (domain.Server, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, name, hostname, public_ip, agent_version, status, last_seen_at, uptime_seconds,
		       cpu_usage_pct, memory_usage_pct, disk_usage_pct, network_rx_mb, network_tx_mb, load_average,
		       os, kernel, cpu_model, cpu_cores, cpu_threads, total_memory_gb, total_disk_gb
		FROM servers
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, serverID)

	server, err := scanServer(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Server{}, ErrNotFound
		}
		return domain.Server{}, err
	}
	return server, nil
}

func (s *PostgresStore) queryServerByID(ctx context.Context, tenantID, serverID string) (domain.Server, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, name, hostname, public_ip, agent_version, status, last_seen_at, uptime_seconds,
		       cpu_usage_pct, memory_usage_pct, disk_usage_pct, network_rx_mb, network_tx_mb, load_average,
		       os, kernel, cpu_model, cpu_cores, cpu_threads, total_memory_gb, total_disk_gb
		FROM servers
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, serverID)

	server, err := scanServer(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Server{}, ErrNotFound
		}
		return domain.Server{}, err
	}
	return server, nil
}

func (s *PostgresStore) queryServicesByServer(ctx context.Context, tenantID, serverID string) ([]domain.Service, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, server_id, name, compose_project, status, container_count, restart_count, published_ports, last_log_timestamp
		FROM service_groups
		WHERE tenant_id = $1 AND server_id = $2
		ORDER BY name ASC
	`, tenantID, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	services := make([]domain.Service, 0)
	for rows.Next() {
		service, scanErr := scanService(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		containers, containerErr := s.queryContainersByService(ctx, service.ID)
		if containerErr != nil {
			return nil, containerErr
		}
		service.Containers = containers
		service.ContainerCount = len(containers)
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *PostgresStore) queryServiceByID(ctx context.Context, tenantID, serviceID string) (domain.Service, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, server_id, name, compose_project, status, container_count, restart_count, published_ports, last_log_timestamp
		FROM service_groups
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, serviceID)

	service, err := scanService(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Service{}, ErrNotFound
		}
		return domain.Service{}, err
	}

	containers, err := s.queryContainersByService(ctx, service.ID)
	if err != nil {
		return domain.Service{}, err
	}
	service.Containers = containers
	service.ContainerCount = len(containers)
	return service, nil
}

func (s *PostgresStore) queryContainersByService(ctx context.Context, serviceID string) ([]domain.Container, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, service_group_id, name, image, status, health, cpu_usage_pct, memory_mb, network_mb,
		       restart_count, uptime, ports, command, last_seen_at
		FROM containers
		WHERE service_group_id = $1
		ORDER BY name ASC
	`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	containers := make([]domain.Container, 0)
	for rows.Next() {
		container, scanErr := scanContainer(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		containers = append(containers, container)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return containers, nil
}

func (s *PostgresStore) queryContainerWithService(ctx context.Context, tenantID, serverID, containerID string) (domain.Service, domain.Container, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT sg.id, sg.tenant_id, sg.server_id, sg.name, sg.compose_project, sg.status, sg.container_count, sg.restart_count,
		       sg.published_ports, sg.last_log_timestamp,
		       c.id, c.service_group_id, c.name, c.image, c.status, c.health, c.cpu_usage_pct, c.memory_mb, c.network_mb,
		       c.restart_count, c.uptime, c.ports, c.command, c.last_seen_at
		FROM containers c
		JOIN service_groups sg ON sg.id = c.service_group_id
		WHERE sg.tenant_id = $1 AND sg.server_id = $2 AND c.id = $3
	`, tenantID, serverID, containerID)

	var (
		service        domain.Service
		container      domain.Container
		servicePorts   []byte
		containerPorts []byte
		lastLog        sql.NullTime
	)
	err := row.Scan(
		&service.ID, &service.TenantID, &service.ServerID, &service.Name, &service.ComposeProject, &service.Status,
		&service.ContainerCount, &service.RestartCount, &servicePorts, &lastLog,
		&container.ID, &container.ServiceID, &container.Name, &container.Image, &container.Status, &container.Health,
		&container.CPUUsagePct, &container.MemoryMB, &container.NetworkMB, &container.RestartCount, &container.Uptime,
		&containerPorts, &container.Command, &container.LastSeenAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Service{}, domain.Container{}, ErrNotFound
		}
		return domain.Service{}, domain.Container{}, err
	}

	service.PublishedPorts = mustUnmarshalStrings(servicePorts)
	if lastLog.Valid {
		service.LastLogTimestamp = lastLog.Time
	}

	container.Ports = mustUnmarshalStrings(containerPorts)
	containers, err := s.queryContainersByService(ctx, service.ID)
	if err != nil {
		return domain.Service{}, domain.Container{}, err
	}
	service.Containers = containers
	service.ContainerCount = len(containers)

	return service, container, nil
}

func (s *PostgresStore) queryServerMetrics(ctx context.Context, serverID string) ([]domain.MetricSeries, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT metric_key, unit, recorded_at, value
		FROM metric_points
		WHERE server_id = $1
		  AND service_group_id IS NULL
		  AND container_id IS NULL
		ORDER BY metric_key ASC, recorded_at ASC
	`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byKey := map[string]domain.MetricSeries{}
	for rows.Next() {
		var (
			key   string
			unit  string
			point domain.MetricPoint
		)
		if err := rows.Scan(&key, &unit, &point.Timestamp, &point.Value); err != nil {
			return nil, err
		}
		series := byKey[key]
		series.Key = key
		series.Unit = unit
		series.Points = append(series.Points, point)
		if len(series.Points) > 30 {
			series.Points = series.Points[len(series.Points)-30:]
		}
		byKey[key] = series
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	series := make([]domain.MetricSeries, 0, len(keys))
	for _, key := range keys {
		series = append(series, byKey[key])
	}
	return series, nil
}

func (s *PostgresStore) queryServerContainerMetricBundle(ctx context.Context, serverID string) (domain.ContainerMetricBundle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, mp.metric_key, mp.recorded_at, mp.value
		FROM metric_points mp
		JOIN containers c ON c.id = mp.container_id
		WHERE mp.server_id = $1
		  AND mp.container_id IS NOT NULL
		ORDER BY mp.metric_key ASC, mp.recorded_at ASC, c.id ASC
	`, serverID)
	if err != nil {
		return domain.ContainerMetricBundle{}, err
	}
	defer rows.Close()

	bundle := domain.ContainerMetricBundle{}
	for rows.Next() {
		var (
			containerID string
			key         string
			recordedAt  time.Time
			value       float64
		)
		if err := rows.Scan(&containerID, &key, &recordedAt, &value); err != nil {
			return domain.ContainerMetricBundle{}, err
		}
		switch key {
		case "cpu_usage_pct":
			bundle.CPU = upsertContainerMetricPoint(bundle.CPU, recordedAt, containerID, value)
		case "memory_mb":
			bundle.Memory = upsertContainerMetricPoint(bundle.Memory, recordedAt, containerID, value)
		case "network_mb":
			bundle.Network = upsertContainerMetricPoint(bundle.Network, recordedAt, containerID, value)
		}
	}
	if err := rows.Err(); err != nil {
		return domain.ContainerMetricBundle{}, err
	}
	return bundle, nil
}

func (s *PostgresStore) queryLogs(ctx context.Context, serviceID, containerID string) ([]domain.LogLine, error) {
	query := `
		SELECT id, server_id, COALESCE(service_group_id, ''), COALESCE(container_id, ''), container_name, service_tag, level, message, occurred_at
		FROM log_lines
		WHERE service_group_id = $1
	`
	args := []any{serviceID}
	if containerID != "" {
		query += ` AND container_id = $2`
		args = append(args, containerID)
	}
	query += ` ORDER BY occurred_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]domain.LogLine, 0)
	for rows.Next() {
		var (
			id              string
			line            domain.LogLine
			storedServiceID string
		)
		if err := rows.Scan(&id, &line.ServerID, &storedServiceID, &line.ContainerID, &line.ContainerName, &line.ServiceTag, &line.Level, &line.Message, &line.Timestamp); err != nil {
			return nil, err
		}
		line.ID = id
		line.ServiceID = storedServiceID
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func (s *PostgresStore) queryEvents(ctx context.Context, serviceID, containerID string) ([]domain.EventLog, error) {
	query := `
		SELECT id, COALESCE(container_id, ''), event_type, message, entity_name, occurred_at
		FROM event_logs
		WHERE service_group_id = $1
	`
	args := []any{serviceID}
	if containerID != "" {
		query += ` AND container_id = $2`
		args = append(args, containerID)
	}
	query += ` ORDER BY occurred_at DESC, id DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.EventLog, 0)
	for rows.Next() {
		var (
			id    string
			event domain.EventLog
		)
		if err := rows.Scan(&id, &event.ContainerID, &event.Type, &event.Message, &event.EntityName, &event.Timestamp); err != nil {
			return nil, err
		}
		event.ID = id
		event.ServiceID = serviceID
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func scanServer(scanner interface{ Scan(dest ...any) error }) (domain.Server, error) {
	var server domain.Server
	err := scanner.Scan(
		&server.ID, &server.TenantID, &server.Name, &server.Hostname, &server.PublicIP, &server.AgentVersion,
		&server.Status, &server.LastSeenAt, &server.UptimeSeconds, &server.CPUUsagePct, &server.MemoryUsagePct,
		&server.DiskUsagePct, &server.NetworkRXMB, &server.NetworkTXMB, &server.LoadAverage, &server.OS,
		&server.Kernel, &server.CPUModel, &server.CPUCores, &server.CPUThreads, &server.TotalMemoryGB, &server.TotalDiskGB,
	)
	return server, err
}

func scanService(scanner interface{ Scan(dest ...any) error }) (domain.Service, error) {
	var (
		service domain.Service
		ports   []byte
		lastLog sql.NullTime
	)
	err := scanner.Scan(&service.ID, &service.TenantID, &service.ServerID, &service.Name, &service.ComposeProject, &service.Status, &service.ContainerCount, &service.RestartCount, &ports, &lastLog)
	if err != nil {
		return domain.Service{}, err
	}
	service.PublishedPorts = mustUnmarshalStrings(ports)
	if lastLog.Valid {
		service.LastLogTimestamp = lastLog.Time
	}
	service.Containers = []domain.Container{}
	return service, nil
}

func scanContainer(scanner interface{ Scan(dest ...any) error }) (domain.Container, error) {
	var (
		container domain.Container
		ports     []byte
	)
	err := scanner.Scan(&container.ID, &container.ServiceID, &container.Name, &container.Image, &container.Status, &container.Health, &container.CPUUsagePct, &container.MemoryMB, &container.NetworkMB, &container.RestartCount, &container.Uptime, &ports, &container.Command, &container.LastSeenAt)
	if err != nil {
		return domain.Container{}, err
	}
	container.Ports = mustUnmarshalStrings(ports)
	return container, nil
}

func (s *PostgresStore) agentByID(ctx context.Context, agentID string) (domain.Agent, error) {
	var agent domain.Agent
	err := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, COALESCE(server_id, ''), name, version, COALESCE(last_seen_at, enrolled_at), enrolled_at, server_name, hostname, description
		FROM agents
		WHERE id = $1
		LIMIT 1
	`, agentID).Scan(
		&agent.ID,
		&agent.TenantID,
		&agent.ServerID,
		&agent.Name,
		&agent.Version,
		&agent.LastSeenAt,
		&agent.EnrolledAt,
		&agent.ServerName,
		&agent.Hostname,
		&agent.Description,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Agent{}, ErrNotFound
		}
		return domain.Agent{}, err
	}
	return agent, nil
}

func (s *PostgresStore) insertSeedContainerMetricBundle(ctx context.Context, tx *sql.Tx, tenantID, serverID string, bundle domain.ContainerMetricBundle, containerIDToServiceID map[string]string) error {
	insertSeries := func(metricKey, unit string, points []domain.ContainerMetricPoint) error {
		for _, point := range points {
			for containerID, value := range point.Values {
				serviceID, ok := containerIDToServiceID[containerID]
				if !ok {
					continue
				}

				if _, err := tx.ExecContext(ctx, `
					INSERT INTO metric_points (id, tenant_id, server_id, service_group_id, container_id, metric_key, unit, recorded_at, value)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				`, mustNewUUIDString(), tenantID, serverID, serviceID, containerID, metricKey, unit, point.Timestamp, value); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := insertSeries("cpu_usage_pct", "%", bundle.CPU); err != nil {
		return err
	}
	if err := insertSeries("memory_mb", "MB", bundle.Memory); err != nil {
		return err
	}
	if err := insertSeries("network_mb", "MB/s", bundle.Network); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) insertContainerMetric(ctx context.Context, tx *sql.Tx, tenantID, serverID, serviceID, containerID, metricKey, unit string, recordedAt time.Time, value float64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO metric_points (id, tenant_id, server_id, service_group_id, container_id, metric_key, unit, recorded_at, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, mustNewUUIDString(), tenantID, serverID, serviceID, containerID, metricKey, unit, recordedAt, value)
	return err
}

func (s *PostgresStore) insertEvent(ctx context.Context, tx *sql.Tx, event storedEvent) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO event_logs (
			id, tenant_id, server_id, service_group_id, container_id, event_type, message, entity_name, occurred_at, created_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, NOW())
	`, event.ID, event.TenantID, event.ServerID, event.ServiceID, event.ContainerID, event.Type, event.Message, event.EntityName, event.Timestamp)
	return err
}

func (s *PostgresStore) createSession(ctx context.Context, tx *sql.Tx, tenantID, userID string) (string, error) {
	token, tokenHash, err := newOpaqueToken()
	if err != nil {
		return "", err
	}

	sessionID, err := newUUIDString()
	if err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_sessions (
			id, tenant_id, user_id, session_token_hash, expires_at, last_seen_at, created_at, user_agent, ip_address
		)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), '', '')
	`, sessionID, tenantID, userID, tokenHash, time.Now().UTC().Add(30*24*time.Hour)); err != nil {
		return "", err
	}

	return token, nil
}

func (s *PostgresStore) deleteStaleServices(ctx context.Context, tx *sql.Tx, serverID string, keep []string) error {
	if len(keep) == 0 {
		_, err := tx.ExecContext(ctx, `DELETE FROM service_groups WHERE server_id = $1`, serverID)
		return err
	}

	_, err := tx.ExecContext(ctx, `
		DELETE FROM service_groups
		WHERE server_id = $1
		  AND NOT (id = ANY($2))
	`, serverID, keep)
	return err
}

func (s *PostgresStore) deleteStaleContainers(ctx context.Context, tx *sql.Tx, serviceID string, keep []string) error {
	if len(keep) == 0 {
		_, err := tx.ExecContext(ctx, `DELETE FROM containers WHERE service_group_id = $1`, serviceID)
		return err
	}

	_, err := tx.ExecContext(ctx, `
		DELETE FROM containers
		WHERE service_group_id = $1
		  AND NOT (id = ANY($2))
	`, serviceID, keep)
	return err
}

func mustUnmarshalStrings(payload []byte) []string {
	values, err := unmarshalStrings(payload)
	if err != nil {
		return nil
	}
	return values
}

func unmarshalStrings(payload []byte) ([]string, error) {
	if len(payload) == 0 {
		return nil, nil
	}

	var values []string
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func marshalStrings(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	return json.Marshal(values)
}

func upsertContainerMetricPoint(points []domain.ContainerMetricPoint, timestamp time.Time, containerKey string, value float64) []domain.ContainerMetricPoint {
	if len(points) > 0 && points[len(points)-1].Timestamp.Equal(timestamp) {
		if points[len(points)-1].Values == nil {
			points[len(points)-1].Values = map[string]float64{}
		}
		points[len(points)-1].Values[containerKey] = value
		return trimContainerMetricPoints(points)
	}

	points = append(points, domain.ContainerMetricPoint{
		Timestamp: timestamp,
		Values: map[string]float64{
			containerKey: value,
		},
	})
	return trimContainerMetricPoints(points)
}

func trimContainerMetricPoints(points []domain.ContainerMetricPoint) []domain.ContainerMetricPoint {
	if len(points) > 30 {
		return points[len(points)-30:]
	}
	return points
}

func newOpaqueToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(bytes)
	return token, hashSecret(token), nil
}

func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func mvpUserRole(role domain.UserRole) string {
	switch role {
	case domain.RoleViewer:
		return string(domain.RoleViewer)
	default:
		return string(domain.RoleAdmin)
	}
}
