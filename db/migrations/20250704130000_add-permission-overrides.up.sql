BEGIN;

-- Store custom permission configurations
CREATE TABLE role_permission_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    canvas_id UUID REFERENCES canvases(id) ON DELETE CASCADE,
    role_name VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID,
    
    -- Ensure either organization_id or canvas_id is set, but not both
    CONSTRAINT check_domain_exclusive CHECK (
        (organization_id IS NOT NULL AND canvas_id IS NULL) OR
        (organization_id IS NULL AND canvas_id IS NOT NULL)
    )
);

-- Store role hierarchy customizations
CREATE TABLE role_hierarchy_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    canvas_id UUID REFERENCES canvases(id) ON DELETE CASCADE,
    child_role VARCHAR(100) NOT NULL,
    parent_role VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID,
    
    -- Ensure either organization_id or canvas_id is set, but not both
    CONSTRAINT check_hierarchy_domain_exclusive CHECK (
        (organization_id IS NOT NULL AND canvas_id IS NULL) OR
        (organization_id IS NULL AND canvas_id IS NOT NULL)
    )
);

-- Create partial unique indexes instead of constraints with WHERE clauses
CREATE UNIQUE INDEX unique_org_permission_override 
    ON role_permission_overrides(organization_id, role_name, resource, action)
    WHERE organization_id IS NOT NULL;

CREATE UNIQUE INDEX unique_canvas_permission_override 
    ON role_permission_overrides(canvas_id, role_name, resource, action)
    WHERE canvas_id IS NOT NULL;

CREATE UNIQUE INDEX unique_org_hierarchy_override 
    ON role_hierarchy_overrides(organization_id, child_role, parent_role)
    WHERE organization_id IS NOT NULL;

CREATE UNIQUE INDEX unique_canvas_hierarchy_override 
    ON role_hierarchy_overrides(canvas_id, child_role, parent_role)
    WHERE canvas_id IS NOT NULL;

-- Add indexes for better query performance
CREATE INDEX idx_role_permission_overrides_org_id ON role_permission_overrides(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_role_permission_overrides_canvas_id ON role_permission_overrides(canvas_id) WHERE canvas_id IS NOT NULL;
CREATE INDEX idx_role_permission_overrides_role_name ON role_permission_overrides(role_name);
CREATE INDEX idx_role_permission_overrides_is_active ON role_permission_overrides(is_active);

CREATE INDEX idx_role_hierarchy_overrides_org_id ON role_hierarchy_overrides(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_role_hierarchy_overrides_canvas_id ON role_hierarchy_overrides(canvas_id) WHERE canvas_id IS NOT NULL;
CREATE INDEX idx_role_hierarchy_overrides_child_role ON role_hierarchy_overrides(child_role);
CREATE INDEX idx_role_hierarchy_overrides_is_active ON role_hierarchy_overrides(is_active);


COMMIT;