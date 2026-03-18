import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { indexDocumentMapper } from "./index_document";
import { onAlertFiresTriggerRenderer } from "./on_alert";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  indexDocument: indexDocumentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFires: onAlertFiresTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  indexDocument: buildActionStateRegistry("indexed"),
};
