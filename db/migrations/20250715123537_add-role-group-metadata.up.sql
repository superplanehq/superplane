BEGIN;

-- Add role_metadata table for storing role display names and descriptions
CREATE TABLE role_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_name VARCHAR(255) NOT NULL,
    domain_type VARCHAR(50) NOT NULL,
    domain_id VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add group_metadata table for storing group display names and descriptions
CREATE TABLE group_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_name VARCHAR(255) NOT NULL,
    domain_type VARCHAR(50) NOT NULL,
    domain_id VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient lookups
CREATE INDEX idx_role_metadata_lookup ON role_metadata (role_name, domain_type, domain_id);
CREATE INDEX idx_group_metadata_lookup ON group_metadata (group_name, domain_type, domain_id);

-- Create unique constraints to prevent duplicate metadata entries
ALTER TABLE role_metadata ADD CONSTRAINT uq_role_metadata_key UNIQUE (role_name, domain_type, domain_id);
ALTER TABLE group_metadata ADD CONSTRAINT uq_group_metadata_key UNIQUE (group_name, domain_type, domain_id);


COMMIT;