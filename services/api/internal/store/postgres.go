package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

// migrationAdvisoryLockID is a stable, application-specific PostgreSQL lock key
// (the ASCII bytes for "XCONTEXT"). It serializes schema migration across
// replicas that start at the same time during a deployment.
const migrationAdvisoryLockID int64 = 0x58434F4E54455854

// ResetForTest clears product tables in an isolated integration-test database.
func (p *Postgres) ResetForTest(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `TRUNCATE context_usage_events,context_redactions,context_objects,context_sessions,context_api_keys CASCADE`)
	return err
}

func (p *Postgres) CommitIngest(session models.Session, isNew bool, obj models.Object, reds []models.Redaction, event models.UsageEvent) error {
	ctx := context.Background()
	tx, e := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if isNew {
		_, e = tx.Exec(ctx, `INSERT INTO context_sessions(id,tenant_id,workspace_id,user_id,name,source,agent,repo,branch,provider,status,token_original,token_returned,token_saved,reduction_percent,estimated_cost_saved,redaction_count,object_count,retrieved_count,created_at,updated_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`, session.ID, session.TenantID, session.WorkspaceID, session.UserID, session.Name, session.Source, session.Agent, session.Repo, session.Branch, session.Provider, session.Status, session.TokenOriginal, session.TokenReturned, session.TokenSaved, session.ReductionPercent, session.EstimatedCostSaved, session.RedactionCount, session.ObjectCount, session.RetrievedCount, session.CreatedAt, session.UpdatedAt, session.ExpiresAt)
	} else {
		tag, updateErr := tx.Exec(ctx, `UPDATE context_sessions SET token_original=token_original+$4,token_returned=token_returned+$5,token_saved=token_saved+$6,estimated_cost_saved=estimated_cost_saved+$7,redaction_count=redaction_count+$8,object_count=object_count+1,reduction_percent=((token_saved+$6)::double precision*100/nullif(token_original+$4,0)),updated_at=$9 WHERE id=$1 AND tenant_id=$2 AND workspace_id=$3`, session.ID, session.TenantID, session.WorkspaceID, obj.TokenOriginal, obj.TokenReturned, obj.TokenSaved, obj.EstimatedCostSaved, obj.RedactionCount, obj.CreatedAt)
		e = updateErr
		if e == nil && tag.RowsAffected() == 0 {
			e = ErrNotFound
		}
	}
	if e != nil {
		return e
	}
	sig, _ := json.Marshal(obj.Signals)
	warn, _ := json.Marshal(obj.Warnings)
	_, e = tx.Exec(ctx, `INSERT INTO context_objects(id,session_id,context_ref,content_type,source,tenant_id,workspace_id,user_id,summary,raw_content,raw_uri,content_hash,compression_version,signals,warnings,token_original,token_returned,token_saved,potential_tokens_saved,delivered_tokens_saved,reduction_percent,estimated_cost_saved,redaction_count,retrieved_count,created_at,expires_at,raw_available) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)`, obj.ID, obj.SessionID, obj.ContextRef, obj.ContentType, obj.Source, obj.TenantID, obj.WorkspaceID, obj.UserID, obj.Summary, obj.RawContent, obj.RawURI, obj.ContentHash, obj.CompressionVersion, sig, warn, obj.TokenOriginal, obj.TokenReturned, obj.TokenSaved, obj.PotentialTokensSaved, obj.DeliveredTokensSaved, obj.ReductionPercent, obj.EstimatedCostSaved, obj.RedactionCount, obj.RetrievedCount, obj.CreatedAt, obj.ExpiresAt, obj.RawAvailable)
	if e != nil {
		return e
	}
	for _, v := range reds {
		if _, e = tx.Exec(ctx, `INSERT INTO context_redactions(id,tenant_id,workspace_id,session_id,object_id,type,source,content_type,count,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, v.ID, v.TenantID, v.WorkspaceID, v.SessionID, v.ObjectID, v.Type, v.Source, v.ContentType, v.Count, v.CreatedAt); e != nil {
			return e
		}
	}
	meta, _ := json.Marshal(event.Metadata)
	_, e = tx.Exec(ctx, `INSERT INTO context_usage_events(id,tenant_id,workspace_id,session_id,object_id,event_type,token_original,token_returned,token_saved,delivered_tokens_saved,metadata,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, event.ID, event.TenantID, event.WorkspaceID, event.SessionID, event.ObjectID, event.EventType, event.TokenOriginal, event.TokenReturned, event.TokenSaved, event.DeliveredTokensSaved, meta, event.CreatedAt)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func OpenPostgres(ctx context.Context, url string) (*Postgres, error) {
	p, e := pgxpool.New(ctx, url)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	return &Postgres{pool: p}, nil
}
func (p *Postgres) Close() { p.pool.Close() }
func (p *Postgres) Migrate(ctx context.Context) error {
	conn, e := p.pool.Acquire(ctx)
	if e != nil {
		return fmt.Errorf("acquire migration connection: %w", e)
	}
	defer conn.Release()

	if _, e = conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationAdvisoryLockID); e != nil {
		return fmt.Errorf("acquire migration lock: %w", e)
	}
	defer func() {
		// Use a fresh context so cancellation of the caller cannot leave the
		// session-level lock held while this pooled connection is released.
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationAdvisoryLockID)
	}()

	entries, e := migrations.Files.ReadDir(".")
	if e != nil {
		return e
	}
	for _, v := range entries {
		b, e := migrations.Files.ReadFile(v.Name())
		if e != nil {
			return e
		}
		if _, e = conn.Exec(ctx, string(b)); e != nil {
			return fmt.Errorf("migration %s: %w", v.Name(), e)
		}
	}
	return nil
}
func dbErr(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return e
}

const sessionCols = `id,tenant_id,workspace_id,coalesce(user_id,''),name,coalesce(source,''),coalesce(agent,''),coalesce(repo,''),coalesce(branch,''),coalesce(provider,''),status,token_original,token_returned,token_saved,reduction_percent,estimated_cost_saved,redaction_count,object_count,retrieved_count,created_at,updated_at,expires_at`

func scanSession(row pgx.Row) (v models.Session, e error) {
	e = row.Scan(&v.ID, &v.TenantID, &v.WorkspaceID, &v.UserID, &v.Name, &v.Source, &v.Agent, &v.Repo, &v.Branch, &v.Provider, &v.Status, &v.TokenOriginal, &v.TokenReturned, &v.TokenSaved, &v.ReductionPercent, &v.EstimatedCostSaved, &v.RedactionCount, &v.ObjectCount, &v.RetrievedCount, &v.CreatedAt, &v.UpdatedAt, &v.ExpiresAt)
	return v, dbErr(e)
}
func (p *Postgres) CreateSession(v models.Session) error {
	_, e := p.pool.Exec(context.Background(), `INSERT INTO context_sessions(id,tenant_id,workspace_id,user_id,name,source,agent,repo,branch,provider,status,token_original,token_returned,token_saved,reduction_percent,estimated_cost_saved,redaction_count,object_count,retrieved_count,created_at,updated_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`, v.ID, v.TenantID, v.WorkspaceID, v.UserID, v.Name, v.Source, v.Agent, v.Repo, v.Branch, v.Provider, v.Status, v.TokenOriginal, v.TokenReturned, v.TokenSaved, v.ReductionPercent, v.EstimatedCostSaved, v.RedactionCount, v.ObjectCount, v.RetrievedCount, v.CreatedAt, v.UpdatedAt, v.ExpiresAt)
	return e
}
func (p *Postgres) UpdateSession(v models.Session) error {
	tag, e := p.pool.Exec(context.Background(), `UPDATE context_sessions SET name=$4,source=$5,agent=$6,repo=$7,branch=$8,provider=$9,status=$10,token_original=$11,token_returned=$12,token_saved=$13,reduction_percent=$14,estimated_cost_saved=$15,redaction_count=$16,object_count=$17,retrieved_count=$18,updated_at=$19,expires_at=$20 WHERE id=$1 AND tenant_id=$2 AND workspace_id=$3`, v.ID, v.TenantID, v.WorkspaceID, v.Name, v.Source, v.Agent, v.Repo, v.Branch, v.Provider, v.Status, v.TokenOriginal, v.TokenReturned, v.TokenSaved, v.ReductionPercent, v.EstimatedCostSaved, v.RedactionCount, v.ObjectCount, v.RetrievedCount, v.UpdatedAt, v.ExpiresAt)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}
func (p *Postgres) GetSession(s models.Scope, id string) (models.Session, error) {
	return scanSession(p.pool.QueryRow(context.Background(), `SELECT `+sessionCols+` FROM context_sessions WHERE id=$1 AND tenant_id=$2 AND workspace_id=$3`, id, s.TenantID, s.WorkspaceID))
}
func (p *Postgres) ListSessions(s models.Scope) []models.Session {
	rows, e := p.pool.Query(context.Background(), `SELECT `+sessionCols+` FROM context_sessions WHERE tenant_id=$1 AND workspace_id=$2 ORDER BY updated_at DESC`, s.TenantID, s.WorkspaceID)
	if e != nil {
		return nil
	}
	defer rows.Close()
	var out []models.Session
	for rows.Next() {
		v, e := scanSession(rows)
		if e == nil {
			out = append(out, v)
		}
	}
	return out
}

const objectCols = `id,session_id,context_ref,content_type,coalesce(source,''),tenant_id,workspace_id,coalesce(user_id,''),summary,coalesce(raw_content,''),coalesce(raw_uri,''),content_hash,compression_version,signals,warnings,token_original,token_returned,token_saved,potential_tokens_saved,delivered_tokens_saved,reduction_percent,estimated_cost_saved,redaction_count,retrieved_count,created_at,expires_at,raw_available`

func scanObject(row pgx.Row) (v models.Object, e error) {
	var sig, warn []byte
	e = row.Scan(&v.ID, &v.SessionID, &v.ContextRef, &v.ContentType, &v.Source, &v.TenantID, &v.WorkspaceID, &v.UserID, &v.Summary, &v.RawContent, &v.RawURI, &v.ContentHash, &v.CompressionVersion, &sig, &warn, &v.TokenOriginal, &v.TokenReturned, &v.TokenSaved, &v.PotentialTokensSaved, &v.DeliveredTokensSaved, &v.ReductionPercent, &v.EstimatedCostSaved, &v.RedactionCount, &v.RetrievedCount, &v.CreatedAt, &v.ExpiresAt, &v.RawAvailable)
	if e == nil {
		_ = json.Unmarshal(sig, &v.Signals)
		_ = json.Unmarshal(warn, &v.Warnings)
	}
	return v, dbErr(e)
}
func (p *Postgres) CreateObject(v models.Object) error {
	sig, _ := json.Marshal(v.Signals)
	warn, _ := json.Marshal(v.Warnings)
	_, e := p.pool.Exec(context.Background(), `INSERT INTO context_objects(id,session_id,context_ref,content_type,source,tenant_id,workspace_id,user_id,summary,raw_content,raw_uri,content_hash,compression_version,signals,warnings,token_original,token_returned,token_saved,potential_tokens_saved,delivered_tokens_saved,reduction_percent,estimated_cost_saved,redaction_count,retrieved_count,created_at,expires_at,raw_available) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)`, v.ID, v.SessionID, v.ContextRef, v.ContentType, v.Source, v.TenantID, v.WorkspaceID, v.UserID, v.Summary, v.RawContent, v.RawURI, v.ContentHash, v.CompressionVersion, sig, warn, v.TokenOriginal, v.TokenReturned, v.TokenSaved, v.PotentialTokensSaved, v.DeliveredTokensSaved, v.ReductionPercent, v.EstimatedCostSaved, v.RedactionCount, v.RetrievedCount, v.CreatedAt, v.ExpiresAt, v.RawAvailable)
	return e
}
func (p *Postgres) UpdateObject(v models.Object) error {
	tag, e := p.pool.Exec(context.Background(), `UPDATE context_objects SET retrieved_count=$4 WHERE id=$1 AND tenant_id=$2 AND workspace_id=$3`, v.ID, v.TenantID, v.WorkspaceID, v.RetrievedCount)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}
func (p *Postgres) GetObject(s models.Scope, id string) (models.Object, error) {
	return scanObject(p.pool.QueryRow(context.Background(), `SELECT `+objectCols+` FROM context_objects WHERE id=$1 AND tenant_id=$2 AND workspace_id=$3`, id, s.TenantID, s.WorkspaceID))
}
func (p *Postgres) GetObjectByRef(s models.Scope, ref string) (models.Object, error) {
	return scanObject(p.pool.QueryRow(context.Background(), `SELECT `+objectCols+` FROM context_objects WHERE context_ref=$1 AND tenant_id=$2 AND workspace_id=$3`, ref, s.TenantID, s.WorkspaceID))
}
func (p *Postgres) ListObjects(s models.Scope, session string) []models.Object {
	q := `SELECT ` + objectCols + ` FROM context_objects WHERE tenant_id=$1 AND workspace_id=$2`
	args := []any{s.TenantID, s.WorkspaceID}
	if session != "" {
		q += ` AND session_id=$3`
		args = append(args, session)
	}
	q += ` ORDER BY created_at DESC`
	rows, e := p.pool.Query(context.Background(), q, args...)
	if e != nil {
		return nil
	}
	defer rows.Close()
	var out []models.Object
	for rows.Next() {
		v, e := scanObject(rows)
		if e == nil {
			out = append(out, v)
		}
	}
	return out
}
func (p *Postgres) AddRedactions(vs []models.Redaction) error {
	if len(vs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, v := range vs {
		batch.Queue(`INSERT INTO context_redactions(id,tenant_id,workspace_id,session_id,object_id,type,source,content_type,count,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, v.ID, v.TenantID, v.WorkspaceID, v.SessionID, v.ObjectID, v.Type, v.Source, v.ContentType, v.Count, v.CreatedAt)
	}
	return p.pool.SendBatch(context.Background(), batch).Close()
}
func (p *Postgres) ListRedactions(s models.Scope, session string) []models.Redaction {
	q := `SELECT id,tenant_id,workspace_id,session_id,object_id,type,coalesce(source,''),coalesce(content_type,''),count,created_at FROM context_redactions WHERE tenant_id=$1 AND workspace_id=$2`
	args := []any{s.TenantID, s.WorkspaceID}
	if session != "" {
		q += ` AND session_id=$3`
		args = append(args, session)
	}
	q += ` ORDER BY created_at DESC`
	rows, e := p.pool.Query(context.Background(), q, args...)
	if e != nil {
		return nil
	}
	defer rows.Close()
	var out []models.Redaction
	for rows.Next() {
		var v models.Redaction
		if rows.Scan(&v.ID, &v.TenantID, &v.WorkspaceID, &v.SessionID, &v.ObjectID, &v.Type, &v.Source, &v.ContentType, &v.Count, &v.CreatedAt) == nil {
			out = append(out, v)
		}
	}
	return out
}
func (p *Postgres) AddEvent(v models.UsageEvent) error {
	meta, _ := json.Marshal(v.Metadata)
	_, e := p.pool.Exec(context.Background(), `INSERT INTO context_usage_events(id,tenant_id,workspace_id,session_id,object_id,event_type,token_original,token_returned,token_saved,delivered_tokens_saved,metadata,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, v.ID, v.TenantID, v.WorkspaceID, v.SessionID, v.ObjectID, v.EventType, v.TokenOriginal, v.TokenReturned, v.TokenSaved, v.DeliveredTokensSaved, meta, v.CreatedAt)
	return e
}
func (p *Postgres) ListEvents(s models.Scope) []models.UsageEvent { return nil }
func (p *Postgres) Search(s models.Scope, session, q string) []models.Object {
	sql := `SELECT ` + objectCols + ` FROM context_objects WHERE tenant_id=$1 AND workspace_id=$2 AND search_document @@ websearch_to_tsquery('simple',$3)`
	args := []any{s.TenantID, s.WorkspaceID, q}
	if session != "" {
		sql += ` AND session_id=$4`
		args = append(args, session)
	}
	sql += ` ORDER BY ts_rank(search_document,websearch_to_tsquery('simple',$3)) DESC LIMIT 50`
	rows, e := p.pool.Query(context.Background(), sql, args...)
	if e != nil {
		return nil
	}
	defer rows.Close()
	var out []models.Object
	for rows.Next() {
		v, e := scanObject(rows)
		if e == nil {
			v.RawContent = ""
			out = append(out, v)
		}
	}
	return out
}
func (p *Postgres) CreateAPIKey(v models.APIKey) error {
	_, e := p.pool.Exec(context.Background(), `INSERT INTO context_api_keys(id,tenant_id,workspace_id,name,key_hash,prefix,scopes,created_by,created_at,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, v.ID, v.TenantID, v.WorkspaceID, v.Name, v.KeyHash, v.Prefix, v.Scopes, v.CreatedBy, v.CreatedAt, v.Status)
	return e
}
func scanKey(row pgx.Row) (v models.APIKey, e error) {
	e = row.Scan(&v.ID, &v.TenantID, &v.WorkspaceID, &v.Name, &v.KeyHash, &v.Prefix, &v.Scopes, &v.CreatedBy, &v.CreatedAt, &v.LastUsedAt, &v.Status)
	return v, dbErr(e)
}
func (p *Postgres) GetAPIKeyByHash(hash string) (models.APIKey, error) {
	return scanKey(p.pool.QueryRow(context.Background(), `SELECT id,tenant_id,workspace_id,name,key_hash,prefix,scopes,coalesce(created_by,''),created_at,last_used_at,status FROM context_api_keys WHERE key_hash=$1 AND status='active'`, hash))
}
func (p *Postgres) ListAPIKeys(s models.Scope) []models.APIKey {
	rows, e := p.pool.Query(context.Background(), `SELECT id,tenant_id,workspace_id,name,'' AS key_hash,prefix,scopes,coalesce(created_by,''),created_at,last_used_at,status FROM context_api_keys WHERE tenant_id=$1 AND workspace_id=$2 ORDER BY created_at DESC`, s.TenantID, s.WorkspaceID)
	if e != nil {
		return nil
	}
	defer rows.Close()
	var out []models.APIKey
	for rows.Next() {
		v, e := scanKey(rows)
		if e == nil {
			out = append(out, v)
		}
	}
	return out
}
func (p *Postgres) RevokeAPIKey(s models.Scope, id string) error {
	tag, e := p.pool.Exec(context.Background(), `UPDATE context_api_keys SET status='revoked' WHERE id=$1 AND tenant_id=$2 AND workspace_id=$3`, id, s.TenantID, s.WorkspaceID)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}
func (p *Postgres) TouchAPIKey(id string, at time.Time) error {
	_, e := p.pool.Exec(context.Background(), `UPDATE context_api_keys SET last_used_at=$2 WHERE id=$1`, id, at)
	return e
}

var _ Store = (*Postgres)(nil)
