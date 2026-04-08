import type { ComponentBaseMapper, CustomFieldRenderer, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { createAnnotationMapper } from "./create_annotation";
import { listAnnotationsMapper } from "./list_annotations";
import { deleteAnnotationMapper } from "./delete_annotation";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryDataSource: queryDataSourceMapper,
  createAnnotation: createAnnotationMapper,
  listAnnotations: listAnnotationsMapper,
  deleteAnnotation: deleteAnnotationMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  queryDataSource: buildActionStateRegistry("queried"),
  createAnnotation: buildActionStateRegistry("created"),
  listAnnotations: buildActionStateRegistry("listed"),
  deleteAnnotation: buildActionStateRegistry("deleted"),
};
