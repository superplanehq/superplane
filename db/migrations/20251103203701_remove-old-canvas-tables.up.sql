BEGIN;

DROP TABLE IF EXISTS
  event_rejections,
  events,
  event_sources,
  stages,
  stage_events,
  stage_event_approvals,
  connections,
  stage_executions,
  connection_groups,
  connection_group_field_sets,
  connection_group_field_set_events,
  resources,
  execution_resources,
  alerts;

COMMIT;