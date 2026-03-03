CREATE TABLE workflow_change_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workflow_id uuid NOT NULL,
    version_id uuid NOT NULL,
    owner_id uuid,
    based_on_version_id uuid,
    status character varying(32) NOT NULL,
    changed_node_ids jsonb DEFAULT '[]'::jsonb NOT NULL,
    conflicting_node_ids jsonb DEFAULT '[]'::jsonb NOT NULL,
    published_at timestamp without time zone,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    CONSTRAINT workflow_change_requests_pkey PRIMARY KEY (id),
    CONSTRAINT workflow_change_requests_workflow_version_key UNIQUE (workflow_id, version_id),
    CONSTRAINT workflow_change_requests_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    CONSTRAINT workflow_change_requests_version_id_fkey FOREIGN KEY (version_id) REFERENCES workflow_versions(id) ON DELETE CASCADE,
    CONSTRAINT workflow_change_requests_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT workflow_change_requests_based_on_version_id_fkey FOREIGN KEY (based_on_version_id) REFERENCES workflow_versions(id) ON DELETE SET NULL
);

CREATE INDEX idx_workflow_change_requests_workflow_id ON workflow_change_requests (workflow_id);
CREATE INDEX idx_workflow_change_requests_status ON workflow_change_requests (workflow_id, status, created_at DESC);
CREATE INDEX idx_workflow_change_requests_owner ON workflow_change_requests (owner_id);
