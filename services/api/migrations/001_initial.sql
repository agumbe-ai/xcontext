CREATE TABLE IF NOT EXISTS context_sessions (
  id text PRIMARY KEY, tenant_id text NOT NULL, workspace_id text NOT NULL, user_id text,
  name text NOT NULL, source text, agent text, repo text, branch text, provider text, status text NOT NULL,
  token_original bigint NOT NULL DEFAULT 0, token_returned bigint NOT NULL DEFAULT 0, token_saved bigint NOT NULL DEFAULT 0,
  reduction_percent double precision NOT NULL DEFAULT 0, estimated_cost_saved double precision NOT NULL DEFAULT 0,
  redaction_count bigint NOT NULL DEFAULT 0, object_count bigint NOT NULL DEFAULT 0, retrieved_count bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, expires_at timestamptz
);
CREATE INDEX IF NOT EXISTS context_sessions_scope_updated_idx ON context_sessions(tenant_id, workspace_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS context_objects (
  id text PRIMARY KEY, tenant_id text NOT NULL, workspace_id text NOT NULL, user_id text, session_id text NOT NULL REFERENCES context_sessions(id) ON DELETE CASCADE,
  context_ref text NOT NULL UNIQUE, content_type text NOT NULL, source text, summary text NOT NULL,
  raw_content text, raw_uri text, content_hash text NOT NULL, compression_version text NOT NULL,
  signals jsonb NOT NULL DEFAULT '[]', warnings jsonb NOT NULL DEFAULT '[]',
  token_original bigint NOT NULL, token_returned bigint NOT NULL, token_saved bigint NOT NULL,
  potential_tokens_saved bigint NOT NULL, delivered_tokens_saved bigint NOT NULL,
  reduction_percent double precision NOT NULL, estimated_cost_saved double precision NOT NULL,
  redaction_count bigint NOT NULL DEFAULT 0, retrieved_count bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL, expires_at timestamptz, raw_available boolean NOT NULL DEFAULT false,
  search_document tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(summary,'') || ' ' || coalesce(raw_content,''))) STORED
);
CREATE INDEX IF NOT EXISTS context_objects_scope_created_idx ON context_objects(tenant_id, workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS context_objects_session_idx ON context_objects(tenant_id, workspace_id, session_id);
CREATE INDEX IF NOT EXISTS context_objects_search_idx ON context_objects USING gin(search_document);

CREATE TABLE IF NOT EXISTS context_redactions (
  id text PRIMARY KEY, tenant_id text NOT NULL, workspace_id text NOT NULL, session_id text NOT NULL, object_id text NOT NULL,
  type text NOT NULL, source text, content_type text, count integer NOT NULL, created_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS context_redactions_scope_idx ON context_redactions(tenant_id, workspace_id, created_at DESC);

CREATE TABLE IF NOT EXISTS context_usage_events (
  id text PRIMARY KEY, tenant_id text NOT NULL, workspace_id text NOT NULL, session_id text, object_id text,
  event_type text NOT NULL, token_original bigint NOT NULL DEFAULT 0, token_returned bigint NOT NULL DEFAULT 0,
  token_saved bigint NOT NULL DEFAULT 0, delivered_tokens_saved bigint NOT NULL DEFAULT 0,
  metadata jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS context_usage_scope_idx ON context_usage_events(tenant_id, workspace_id, created_at DESC);

CREATE TABLE IF NOT EXISTS context_api_keys (
  id text PRIMARY KEY, tenant_id text NOT NULL, workspace_id text NOT NULL, name text NOT NULL,
  key_hash text NOT NULL UNIQUE, prefix text NOT NULL, scopes text[] NOT NULL, created_by text,
  created_at timestamptz NOT NULL, last_used_at timestamptz, status text NOT NULL
);
CREATE INDEX IF NOT EXISTS context_api_keys_scope_idx ON context_api_keys(tenant_id, workspace_id, created_at DESC);

