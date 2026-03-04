import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import {
  onReconciliationCompletedCustomFieldRenderer,
  onReconciliationCompletedTriggerRenderer,
} from "./on_reconciliation_completed";
import { reconcileSourceMapper } from "./reconcile_source";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  reconcileSource: reconcileSourceMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onReconciliationCompleted: onReconciliationCompletedTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onReconciliationCompleted: onReconciliationCompletedCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  reconcileSource: buildActionStateRegistry("Reconciled"),
};
