import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry, CustomFieldRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";

import { createEventMapper } from "./create_event";
import { onAlertFiredTriggerRenderer } from "./on_alert_fired";
import { honeycombCustomFieldRenderers } from "./custom_fields";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createEvent: createEventMapper,
  "honeycomb.createEvent": createEventMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFired: onAlertFiredTriggerRenderer,
  "honeycomb.onAlertFired": onAlertFiredTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createEvent: buildActionStateRegistry("Sent"),
  "honeycomb.createEvent": buildActionStateRegistry("Sent"),
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  ...honeycombCustomFieldRenderers,
};
