import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIncidentTriggerRenderer } from "./on_incident";
import { onIncidentStatusUpdateTriggerRenderer } from "./on_incident_status_update";
import { onIncidentAnnotatedTriggerRenderer } from "./on_incident_annotated";
import { createIncidentMapper } from "./create_incident";
import { updateIncidentMapper } from "./update_incident";
import { acknowledgeIncidentMapper } from "./acknowledge_incident";
import { resolveIncidentMapper } from "./resolve_incident";
import { escalateIncidentMapper } from "./escalate_incident";
import { reassignEscalationPolicyMapper } from "./reassign_escalation_policy";
import { annotateIncidentMapper } from "./annotate_incident";
import { listIncidentsMapper, LIST_INCIDENTS_STATE_REGISTRY } from "./list_incidents";
import { snoozeIncidentMapper } from "./snooze_incident";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  updateIncident: updateIncidentMapper,
  acknowledgeIncident: acknowledgeIncidentMapper,
  resolveIncident: resolveIncidentMapper,
  escalateIncident: escalateIncidentMapper,
  reassignEscalationPolicy: reassignEscalationPolicyMapper,
  annotateIncident: annotateIncidentMapper,
  listIncidents: listIncidentsMapper,
  snoozeIncident: snoozeIncidentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIncident: onIncidentTriggerRenderer,
  onIncidentStatusUpdate: onIncidentStatusUpdateTriggerRenderer,
  onIncidentAnnotated: onIncidentAnnotatedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  updateIncident: buildActionStateRegistry("updated"),
  acknowledgeIncident: buildActionStateRegistry("acknowledged"),
  resolveIncident: buildActionStateRegistry("resolved"),
  escalateIncident: buildActionStateRegistry("escalated"),
  reassignEscalationPolicy: buildActionStateRegistry("reassigned"),
  annotateIncident: buildActionStateRegistry("annotated"),
  listIncidents: LIST_INCIDENTS_STATE_REGISTRY,
  snoozeIncident: buildActionStateRegistry("snoozed"),
};
