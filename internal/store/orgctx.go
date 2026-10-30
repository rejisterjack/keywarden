package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/rejisterjack/keywarden/internal/fail"
)

type orgKey struct{}

// DefaultOrgID is the uuid backfilled by migration 004 for single-tenant stacks.
var DefaultOrgID = uuid.MustParse("00000000-0000-4000-8000-000000000001")

// WithOrg stores the active organization on ctx for store queries.
func WithOrg(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, orgKey{}, orgID)
}

// OrgFrom returns the organization bound to ctx, if any.
func OrgFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(orgKey{}).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

func (s *Store) org(ctx context.Context) (uuid.UUID, error) {
	if id, ok := OrgFrom(ctx); ok {
		return id, nil
	}
	if s.singleTenant {
		return s.defaultOrgID, nil
	}
	return uuid.Nil, fail.New(401, "unauthorized")
}
