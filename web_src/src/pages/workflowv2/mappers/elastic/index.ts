import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { indexDocumentMapper } from "./index_document";
import { createCaseMapper } from "./create_case";
import { getCaseMapper } from "./get_case";
import { updateCaseMapper } from "./update_case";
import { onAlertFiresTriggerRenderer } from "./on_alert";
import { onCaseStatusChangeTriggerRenderer } from "./on_case_status_change";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  indexDocument: indexDocumentMapper,
  createCase: createCaseMapper,
  getCase: getCaseMapper,
  updateCase: updateCaseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFires: onAlertFiresTriggerRenderer,
  onCaseStatusChange: onCaseStatusChangeTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  indexDocument: buildActionStateRegistry("indexed"),
  createCase: buildActionStateRegistry("created"),
  getCase: buildActionStateRegistry("retrieved"),
  updateCase: buildActionStateRegistry("updated"),
};
