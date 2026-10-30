package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/slug"
)

// Organization is one tenant.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// EnsureDefaultOrg loads the migration-seeded default organization id.
func (s *Store) EnsureDefaultOrg(ctx context.Context) error {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM organizations WHERE slug = 'default'`).Scan(&id)
	if err != nil {
		return fmt.Errorf("default organization: %w", err)
	}
	s.defaultOrgID = id
	return nil
}

// CreateOrganization inserts a tenant. Used in multi-tenant mode.
func (s *Store) CreateOrganization(ctx context.Context, slugID, name string) (Organization, error) {
	if !slug.OK(slugID) {
		return Organization{}, fail.New(400, "organization slug must be a lowercase slug")
	}
	if name == "" || len(name) > 200 {
		return Organization{}, fail.New(400, "organization name is required")
	}
	var org Organization
	err := s.pool.QueryRow(ctx, `
		INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)
		RETURNING id, slug, name, created_at`, uuid.New(), slugID, name,
	).Scan(&org.ID, &org.Slug, &org.Name, &org.CreatedAt)
	if err != nil {
		return Organization{}, mapWrite(err, "organization already exists")
	}
	return org, nil
}

// GetOrganization loads one tenant by id.
func (s *Store) GetOrganization(ctx context.Context, id uuid.UUID) (Organization, error) {
	var org Organization
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, created_at FROM organizations WHERE id = $1`, id,
	).Scan(&org.ID, &org.Slug, &org.Name, &org.CreatedAt)
	if err != nil {
		return Organization{}, mapRead(err, "organization not found")
	}
	return org, nil
}

// IssueOrgAdminToken returns a plaintext bearer once. The hash is stored.
func (s *Store) IssueOrgAdminToken(ctx context.Context, orgID uuid.UUID, label string) (string, error) {
	if _, err := s.GetOrganization(ctx, orgID); err != nil {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	plain := "kwa_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	if label == "" {
		label = "admin"
	}
	if len(label) > 200 {
		return "", fail.New(400, "token label is too long")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO org_admin_tokens (id, org_id, token_hash, label)
		VALUES ($1, $2, $3, $4)`, uuid.New(), orgID, sum[:], label)
	if err != nil {
		return "", err
	}
	return plain, nil
}

// AuthenticateAdmin resolves an org admin bearer token.
// In single-tenant mode the process KEYWARDEN_ADMIN_TOKEN is also accepted
// and maps to the default organization.
func (s *Store) AuthenticateAdmin(ctx context.Context, bearer, envToken string) (uuid.UUID, error) {
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return uuid.Nil, fail.New(401, "unauthorized")
	}
	if s.singleTenant && envToken != "" {
		a := sha256.Sum256([]byte(bearer))
		b := sha256.Sum256([]byte(envToken))
		if subtle.ConstantTimeCompare(a[:], b[:]) == 1 {
			return s.defaultOrgID, nil
		}
	}
	sum := sha256.Sum256([]byte(bearer))
	var orgID uuid.UUID
	var revoked *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT org_id, revoked_at FROM org_admin_tokens WHERE token_hash = $1`, sum[:],
	).Scan(&orgID, &revoked)
	if err != nil {
		return uuid.Nil, fail.New(401, "unauthorized")
	}
	if revoked != nil {
		return uuid.Nil, fail.New(401, "unauthorized")
	}
	return orgID, nil
}

// DefaultOrg returns the single-tenant organization id.
func (s *Store) DefaultOrg() uuid.UUID { return s.defaultOrgID }

// SingleTenant reports whether the process uses the env admin token.
func (s *Store) SingleTenant() bool { return s.singleTenant }
