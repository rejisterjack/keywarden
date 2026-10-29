package approval

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rejisterjack/keywarden/internal/store"
)

// Wait polls until the approval is no longer pending.
// Cancellation expires a still-pending approval and does not mean the call may proceed.
func Wait(ctx context.Context, st *store.Store, id uuid.UUID) (string, error) {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		current, err := st.GetApproval(ctx, id)
		if err != nil {
			if ctx.Err() != nil {
				status, expErr := st.ExpireApproval(context.Background(), id, "request cancelled")
				if expErr != nil {
					return "", expErr
				}
				return status, nil
			}
			return "", err
		}
		if current.Status != "pending" {
			return current.Status, nil
		}
		if !time.Now().Before(current.ExpiresAt) {
			return st.ExpireApproval(context.Background(), id, "deadline passed")
		}
		select {
		case <-ctx.Done():
			return st.ExpireApproval(context.Background(), id, "request cancelled")
		case <-tick.C:
		}
	}
}
