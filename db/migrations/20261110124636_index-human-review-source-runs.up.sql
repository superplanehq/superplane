CREATE INDEX CONCURRENTLY factory_human_review_source_run
    ON factory_human_review_decisions (source_run_id)
    WHERE source_run_id IS NOT NULL;
