import type { ComponentBaseMapper, CustomFieldRenderer, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getSilenceEventStateRegistry } from "./get_silence_state";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { listSilencesMapper } from "./list_silences";
import { getSilenceMapper } from "./get_silence";
import { createSilenceMapper } from "./create_silence";
import { deleteSilenceMapper } from "./delete_silence";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryDataSource: queryDataSourceMapper,
  createSilence: createSilenceMapper,
  deleteSilence: deleteSilenceMapper,
  getSilence: getSilenceMapper,
  listSilences: listSilencesMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  queryDataSource: buildActionStateRegistry("queried"),
  createSilence: buildActionStateRegistry("created"),
  deleteSilence: buildActionStateRegistry("deleted"),
  getSilence: getSilenceEventStateRegistry,
  listSilences: buildActionStateRegistry("listed"),
};
