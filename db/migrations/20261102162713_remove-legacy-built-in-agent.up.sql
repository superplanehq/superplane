-- Capture hosted resource identifiers and stop old API/worker processes first.
DROP TABLE agent_session_messages;
DROP TABLE agent_sessions;
ALTER TABLE workflows DROP COLUMN dismissed_agent_suggestion_ids;
UPDATE organizations
SET enabled_experimental_features = enabled_experimental_features - 'claude_managed_agents'
WHERE enabled_experimental_features ? 'claude_managed_agents';
DELETE FROM casbin_rule WHERE ptype = 'p' AND v2 = 'agents';
