ALTER TABLE factory_velocity_syncs ADD COLUMN scope_repository text NOT NULL DEFAULT '';
UPDATE factory_velocity_syncs SET scope_repository = lower(trim(repository));
ALTER TABLE factory_velocity_syncs DROP CONSTRAINT factory_velocity_syncs_pkey;
ALTER TABLE factory_velocity_syncs ADD PRIMARY KEY (factory_id, scope_repository);
