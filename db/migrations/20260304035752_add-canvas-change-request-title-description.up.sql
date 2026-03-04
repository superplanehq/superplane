ALTER TABLE workflow_change_requests
ADD COLUMN title text NOT NULL DEFAULT '',
ADD COLUMN description text NOT NULL DEFAULT '';
