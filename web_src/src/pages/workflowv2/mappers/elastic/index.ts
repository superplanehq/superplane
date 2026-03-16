import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { indexDocumentMapper } from "./index_document";
import { getDocumentMapper } from "./get_document";
import { updateDocumentMapper } from "./update_document";
import { onAlertFiresTriggerRenderer } from "./on_alert";
import { onDocumentIndexedTriggerRenderer } from "./on_document_indexed";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  indexDocument: indexDocumentMapper,
  getDocument: getDocumentMapper,
  updateDocument: updateDocumentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFires: onAlertFiresTriggerRenderer,
  onDocumentIndexed: onDocumentIndexedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  indexDocument: buildActionStateRegistry("indexed"),
  getDocument: buildActionStateRegistry("retrieved"),
  updateDocument: buildActionStateRegistry("updated"),
};
