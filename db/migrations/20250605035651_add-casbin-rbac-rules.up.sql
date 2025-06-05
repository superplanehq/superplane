BEGIN;

-- Migration: Create Casbin policy table
-- This table stores all RBAC policies and role assignments

CREATE TABLE IF NOT EXISTS casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(100),
    v1 VARCHAR(100),
    v2 VARCHAR(100),
    v3 VARCHAR(100),
    v4 VARCHAR(100),
    v5 VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_casbin_rule_ptype ON casbin_rule(ptype);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v0 ON casbin_rule(v0);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v1 ON casbin_rule(v1);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v2 ON casbin_rule(v2);

-- TODO: Verify if organization level rules should be inserted
-- Insert default organization-level policies
-- Organization Owner permissions
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'org:owner', 'organization', 'admin'),
    ('p', 'org:owner', 'canvas', 'admin'),
    ('p', 'org:owner', 'user', 'admin'),
    ('p', 'org:owner', 'secret', 'admin');

-- Organization Admin permissions  
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'org:admin', 'canvas', 'admin'),
    ('p', 'org:admin', 'user', 'write'),
    ('p', 'org:admin', 'secret', 'write'),
    ('p', 'org:admin', 'organization', 'read');

-- Organization Member permissions
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'org:member', 'organization', 'read'),
    ('p', 'org:member', 'canvas', 'read');

-- Canvas-level policies
-- Canvas Owner permissions
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'canvas:owner', 'canvas', 'admin'),
    ('p', 'canvas:owner', 'stage', 'admin'),
    ('p', 'canvas:owner', 'execution', 'admin'),
    ('p', 'canvas:owner', 'event_source', 'admin');

-- Canvas Admin permissions
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'canvas:admin', 'canvas', 'write'),
    ('p', 'canvas:admin', 'stage', 'write'),
    ('p', 'canvas:admin', 'execution', 'write'),
    ('p', 'canvas:admin', 'event_source', 'write');

-- Canvas Developer permissions
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'canvas:developer', 'canvas', 'read'),
    ('p', 'canvas:developer', 'stage', 'write'),
    ('p', 'canvas:developer', 'execution', 'create'),
    ('p', 'canvas:developer', 'event_source', 'read');

-- Canvas Contributor permissions
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'canvas:contributor', 'canvas', 'read'),
    ('p', 'canvas:contributor', 'stage', 'read'),
    ('p', 'canvas:contributor', 'execution', 'create'),
    ('p', 'canvas:contributor', 'event_source', 'read');

-- Canvas Viewer permissions
INSERT INTO casbin_rule(ptype, v0, v1, v2) VALUES
    ('p', 'canvas:viewer', 'canvas', 'read'),
    ('p', 'canvas:viewer', 'stage', 'read'),
    ('p', 'canvas:viewer', 'execution', 'read'),
    ('p', 'canvas:viewer', 'event_source', 'read');

-- Role inheritance (g = grouping policy)
-- Organization role hierarchy
INSERT INTO casbin_rule(ptype, v0, v1) VALUES
    ('g', 'org:owner', 'org:admin'),
    ('g', 'org:admin', 'org:member');

-- Canvas role hierarchy  
INSERT INTO casbin_rule(ptype, v0, v1) VALUES
    ('g', 'canvas:owner', 'canvas:admin'),
    ('g', 'canvas:admin', 'canvas:developer'),
    ('g', 'canvas:developer', 'canvas:contributor'),
    ('g', 'canvas:contributor', 'canvas:viewer');

-- Example of how users are assigned to roles
-- These are examples - replace with actual user IDs
-- INSERT INTO casbin_rule(ptype, v0, v1) VALUES
--     ('g', 'user:alice@example.com', 'org:owner'),
--     ('g', 'user:bob@example.com', 'org:admin'),
--     ('g', 'user:charlie@example.com', 'org:member');

-- Example canvas-specific role assignments
-- INSERT INTO casbin_rule(ptype, v0, v1) VALUES
--     ('g', 'user:alice@example.com', 'canvas:owner:canvas-uuid-1'),
--     ('g', 'user:bob@example.com', 'canvas:developer:canvas-uuid-1'),
--     ('g', 'user:charlie@example.com', 'canvas:viewer:canvas-uuid-1');

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_casbin_rule_updated_at
    BEFORE UPDATE ON casbin_rule
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE casbin_rule IS 'Casbin RBAC policies and role assignments';
COMMENT ON COLUMN casbin_rule.ptype IS 'Policy type: p for policy, g for grouping (roles)';
COMMENT ON COLUMN casbin_rule.v0 IS 'Subject (user or role)';
COMMENT ON COLUMN casbin_rule.v1 IS 'Object (resource) or parent role';
COMMENT ON COLUMN casbin_rule.v2 IS 'Action (permission) - only for policy type p';

COMMIT;