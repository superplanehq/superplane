ALTER TABLE public.factory_velocity_syncs
    RENAME COLUMN rules_generation TO sync_state;

UPDATE public.factory_velocity_syncs SET sync_state = 0 WHERE sync_state > 0;
