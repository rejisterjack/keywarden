package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rejisterjack/keywarden/internal/canonical"
)

const auditLock int64 = 7483926150234

func (s *Store) AppendAudit(ctx context.Context, eventType string, payload map[string]any) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := appendAuditTx(ctx, tx, eventType, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func appendAuditTx(ctx context.Context, tx pgx.Tx, eventType string, payload map[string]any) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auditLock); err != nil {
		return err
	}
	var prev []byte
	err := tx.QueryRow(ctx, `SELECT hash FROM audit_events ORDER BY seq DESC LIMIT 1`).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		prev = make([]byte, 32)
		err = nil
	}
	if err != nil {
		return err
	}
	body := make(map[string]any, len(payload)+2)
	for k, v := range payload {
		body[k] = v
	}
	body["event_type"] = eventType
	body["at"] = time.Now().UTC().Format(time.RFC3339Nano)
	canon, err := canonical.Bytes(body)
	if err != nil {
		return err
	}
	mac := sha256.New()
	_, _ = mac.Write(prev)
	_, _ = mac.Write(canon)
	sum := mac.Sum(nil)
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (prev_hash, hash, event_type, payload)
		VALUES ($1, $2, $3, $4)`, prev, sum, eventType, string(canon))
	return err
}

func (s *Store) ListAudit(ctx context.Context, after int64, limit int) ([]AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT seq, event_type, payload, created_at
		FROM audit_events
		WHERE seq > $1
		ORDER BY seq
		LIMIT $2`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEvent
	for rows.Next() {
		var ev AuditEvent
		var payload string
		if err := rows.Scan(&ev.Seq, &ev.EventType, &payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.Payload = json.RawMessage(payload)
		out = append(out, ev)
	}
	if out == nil {
		out = []AuditEvent{}
	}
	return out, rows.Err()
}

func (s *Store) VerifyAudit(ctx context.Context) (VerifyResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT seq, prev_hash, hash, event_type, payload
		FROM audit_events
		ORDER BY seq`)
	if err != nil {
		return VerifyResult{}, err
	}
	defer rows.Close()
	var prev []byte
	var n int
	var head []byte
	for rows.Next() {
		var seq int64
		var prevHash, hash []byte
		var eventType, payload string
		if err := rows.Scan(&seq, &prevHash, &hash, &eventType, &payload); err != nil {
			return VerifyResult{}, err
		}
		n++
		if seq != int64(n) {
			return VerifyResult{OK: false, Events: n, BrokenSeq: seq, Reason: "sequence gap"}, nil
		}
		if n == 1 {
			if len(prevHash) != 32 || !allZero(prevHash) {
				return VerifyResult{OK: false, Events: n, BrokenSeq: seq, Reason: "genesis hash is not zero"}, nil
			}
		} else if !bytesEqual(prevHash, prev) {
			return VerifyResult{OK: false, Events: n, BrokenSeq: seq, Reason: "previous hash does not match"}, nil
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
			return VerifyResult{OK: false, Events: n, BrokenSeq: seq, Reason: "payload is not JSON"}, nil
		}
		if decoded["event_type"] != eventType {
			return VerifyResult{OK: false, Events: n, BrokenSeq: seq, Reason: "event type does not match payload"}, nil
		}
		mac := sha256.New()
		_, _ = mac.Write(prevHash)
		_, _ = mac.Write([]byte(payload))
		sum := mac.Sum(nil)
		if !bytesEqual(sum, hash) {
			return VerifyResult{OK: false, Events: n, BrokenSeq: seq, Reason: "hash mismatch"}, nil
		}
		prev = hash
		head = hash
	}
	if err := rows.Err(); err != nil {
		return VerifyResult{}, err
	}
	res := VerifyResult{OK: true, Events: n}
	if head != nil {
		res.Head = hex.EncodeToString(head)
	}
	return res, nil
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
