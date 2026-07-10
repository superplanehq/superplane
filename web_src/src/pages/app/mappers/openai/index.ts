import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { getUsageMapper } from "./get_usage";
import { getFileMapper } from "./get_file";
import { downloadFileMapper } from "./download_file";
import { downloadContainerFileMapper } from "./download_container_file";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
  getUsage: getUsageMapper,
  getFile: getFileMapper,
  downloadFile: downloadFileMapper,
  downloadContainerFile: downloadContainerFileMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
  getUsage: buildActionStateRegistry("completed"),
  getFile: buildActionStateRegistry("fetched"),
  downloadFile: buildActionStateRegistry("downloaded"),
  downloadContainerFile: buildActionStateRegistry("downloaded"),
};
