import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { indexDocumentMapper } from "./index_document";
import { createCaseMapper } from "./create_case";
import { getCaseMapper } from "./get_case";
import { updateCaseMapper } from "./update_case";
import { onCaseStatusChangeTriggerRenderer } from "./on_case_status_change";
import { getDocumentMapper } from "./get_document";
import { updateDocumentMapper } from "./update_document";
import { onAlertFiresTriggerRenderer } from "./on_alert";
import { onDocumentIndexedTriggerRenderer } from "./on_document_indexed";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  indexDocument: indexDocumentMapper,
  createCase: createCaseMapper,
  getCase: getCaseMapper,
  updateCase: updateCaseMapper,
  getDocument: getDocumentMapper,
  updateDocument: updateDocumentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFires: onAlertFiresTriggerRenderer,
  onCaseStatusChange: onCaseStatusChangeTriggerRenderer,
  onDocumentIndexed: onDocumentIndexedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  indexDocument: buildActionStateRegistry("indexed"),
  createCase: buildActionStateRegistry("created"),
  getCase: buildActionStateRegistry("retrieved"),
  updateCase: buildActionStateRegistry("updated"),
  getDocument: buildActionStateRegistry("retrieved"),
  updateDocument: buildActionStateRegistry("updated"),
};
