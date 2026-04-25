BEGIN;

-- Shared trigger function for update_time columns.
CREATE OR REPLACE FUNCTION janus_set_update_time()
RETURNS TRIGGER AS $$
BEGIN
  NEW.update_time = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ------------------------
-- Account and auth tables
-- ------------------------

CREATE TABLE IF NOT EXISTS janus_auth_organization (
  organization_id BIGSERIAL PRIMARY KEY,
  organization_name TEXT NOT NULL UNIQUE,
  create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS janus_auth_team (
  team_id BIGSERIAL PRIMARY KEY,
  team_name TEXT NOT NULL UNIQUE,
  model_list TEXT NOT NULL DEFAULT '*',
  organization_id BIGINT NOT NULL REFERENCES janus_auth_organization(organization_id) ON DELETE RESTRICT,
  create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS janus_admin_user (
  admin_user_id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS janus_auth_key (
  key_id BIGSERIAL PRIMARY KEY,
  key_content TEXT NOT NULL UNIQUE,
  key_name TEXT NOT NULL,
  -- Kept as comma-separated string for compatibility with current code.
  model_list TEXT NOT NULL DEFAULT '*',
  team_id BIGINT NOT NULL REFERENCES janus_auth_team(team_id) ON DELETE RESTRICT,
  organization_id BIGINT NOT NULL REFERENCES janus_auth_organization(organization_id) ON DELETE RESTRICT,
  balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
  total_spend NUMERIC(20, 8) NOT NULL DEFAULT 0,
  request_per_minute INTEGER NOT NULL DEFAULT 0 CHECK (request_per_minute >= 0),
  spend_limit_per_week NUMERIC(20, 8) NOT NULL DEFAULT 0,
  create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expire_time TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_auth_key_team_id ON janus_auth_key (team_id);
CREATE INDEX IF NOT EXISTS idx_auth_key_org_id ON janus_auth_key (organization_id);
CREATE INDEX IF NOT EXISTS idx_auth_key_expire_time ON janus_auth_key (expire_time);
CREATE INDEX IF NOT EXISTS idx_auth_key_balance ON janus_auth_key (balance);
CREATE INDEX IF NOT EXISTS idx_admin_user_username_enabled ON janus_admin_user (username, enabled);

-- ------------------------
-- Model config tables
-- ------------------------

CREATE TABLE IF NOT EXISTS janus_model_group (
  group_id BIGSERIAL PRIMARY KEY,
  group_name TEXT NOT NULL UNIQUE,
  strategy TEXT NOT NULL DEFAULT 'round-robin'
    CHECK (strategy IN ('round-robin', 'weighted', 'least-inflight', 'latency-based')),
  cost_per_input_token NUMERIC(20, 10) NOT NULL DEFAULT 0,
  cost_per_output_token NUMERIC(20, 10) NOT NULL DEFAULT 0,
  request_defaults JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS janus_model_endpoint (
  endpoint_id BIGSERIAL PRIMARY KEY,
  group_id BIGINT NOT NULL REFERENCES janus_model_group(group_id) ON DELETE CASCADE,
  endpoint_name TEXT NOT NULL,
  provider_type TEXT NOT NULL,
  upstream_model_name TEXT NOT NULL,
  base_url TEXT NOT NULL,
  -- Use secret ref in production (k8s secret/config center), avoid plaintext key.
  api_key_secret_ref TEXT,
  weight INTEGER NOT NULL DEFAULT 100 CHECK (weight > 0),
  timeout_seconds INTEGER NOT NULL DEFAULT 60 CHECK (timeout_seconds > 0),
  retry_times INTEGER NOT NULL DEFAULT 1 CHECK (retry_times >= 0),
  skip_tls_verify BOOLEAN NOT NULL DEFAULT FALSE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (group_id, endpoint_name)
);

CREATE INDEX IF NOT EXISTS idx_model_endpoint_group_enabled
  ON janus_model_endpoint (group_id, enabled);

-- ------------------------
-- Billing and usage tables
-- ------------------------

CREATE TABLE IF NOT EXISTS janus_spend_log (
  record_id BIGSERIAL PRIMARY KEY,
  request_id TEXT NOT NULL,
  key_id BIGINT NOT NULL REFERENCES janus_auth_key(key_id) ON DELETE RESTRICT,
  key_content TEXT NOT NULL,
  team_id BIGINT NOT NULL REFERENCES janus_auth_team(team_id) ON DELETE RESTRICT,
  organization_id BIGINT NOT NULL REFERENCES janus_auth_organization(organization_id) ON DELETE RESTRICT,
  model_group TEXT NOT NULL,
  spend NUMERIC(20, 8) NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  prompt_tokens INTEGER NOT NULL DEFAULT 0,
  completion_tokens INTEGER NOT NULL DEFAULT 0,
  create_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_spend_log_create_time ON janus_spend_log (create_time);
CREATE INDEX IF NOT EXISTS idx_spend_log_key_id ON janus_spend_log (key_id);
CREATE INDEX IF NOT EXISTS idx_spend_log_team_time ON janus_spend_log (team_id, create_time);
CREATE INDEX IF NOT EXISTS idx_spend_log_org_time ON janus_spend_log (organization_id, create_time);

-- Optional summary table for faster dashboard query.
CREATE TABLE IF NOT EXISTS janus_key_spend_daily (
  summary_date DATE NOT NULL,
  key_id BIGINT NOT NULL REFERENCES janus_auth_key(key_id) ON DELETE RESTRICT,
  key_content TEXT NOT NULL,
  organization_id BIGINT NOT NULL REFERENCES janus_auth_organization(organization_id) ON DELETE RESTRICT,
  total_spend NUMERIC(20, 8) NOT NULL DEFAULT 0,
  total_tokens BIGINT NOT NULL DEFAULT 0,
  request_count BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (summary_date, key_id)
);

-- ------------------------
-- Triggers for update_time
-- ------------------------

DROP TRIGGER IF EXISTS trg_org_update_time ON janus_auth_organization;
CREATE TRIGGER trg_org_update_time
BEFORE UPDATE ON janus_auth_organization
FOR EACH ROW
EXECUTE FUNCTION janus_set_update_time();

DROP TRIGGER IF EXISTS trg_team_update_time ON janus_auth_team;
CREATE TRIGGER trg_team_update_time
BEFORE UPDATE ON janus_auth_team
FOR EACH ROW
EXECUTE FUNCTION janus_set_update_time();

DROP TRIGGER IF EXISTS trg_admin_user_update_time ON janus_admin_user;
CREATE TRIGGER trg_admin_user_update_time
BEFORE UPDATE ON janus_admin_user
FOR EACH ROW
EXECUTE FUNCTION janus_set_update_time();

DROP TRIGGER IF EXISTS trg_key_update_time ON janus_auth_key;
CREATE TRIGGER trg_key_update_time
BEFORE UPDATE ON janus_auth_key
FOR EACH ROW
EXECUTE FUNCTION janus_set_update_time();

DROP TRIGGER IF EXISTS trg_model_group_update_time ON janus_model_group;
CREATE TRIGGER trg_model_group_update_time
BEFORE UPDATE ON janus_model_group
FOR EACH ROW
EXECUTE FUNCTION janus_set_update_time();

DROP TRIGGER IF EXISTS trg_model_endpoint_update_time ON janus_model_endpoint;
CREATE TRIGGER trg_model_endpoint_update_time
BEFORE UPDATE ON janus_model_endpoint
FOR EACH ROW
EXECUTE FUNCTION janus_set_update_time();

COMMIT;
