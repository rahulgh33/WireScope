-- Add org_id to main tables for tenant isolation

ALTER TABLE events_seen ADD COLUMN org_id VARCHAR(64) DEFAULT 'default';
ALTER TABLE agg_1m ADD COLUMN org_id VARCHAR(64) DEFAULT 'default';
ALTER TABLE diagnosis_history ADD COLUMN org_id VARCHAR(64) DEFAULT 'default';

-- Update indexes to include org_id
CREATE INDEX idx_agg_1m_org ON agg_1m(org_id, client_id, target, window_start_ts);
CREATE INDEX idx_diagnosis_org ON diagnosis_history(org_id, created_at);

-- Organizations table
CREATE TABLE organizations (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    quota_events_per_hour INT DEFAULT 10000,
    quota_clients INT DEFAULT 100
);

-- Users table
CREATE TABLE users (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) REFERENCES organizations(id),
    email VARCHAR(255) NOT NULL UNIQUE,
    role VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO organizations (id, name) VALUES ('default', 'Default Organization');
