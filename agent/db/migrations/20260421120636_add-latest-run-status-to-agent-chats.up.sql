ALTER TABLE agent_chats
    ADD COLUMN latest_run_status TEXT NOT NULL DEFAULT '';
