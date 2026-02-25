import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createSandbox: baseMapper,
  getPreviewUrl: baseMapper,
  executeCode: baseMapper,
  executeCommand: baseMapper,
  deleteSandbox: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createSandbox: buildActionStateRegistry("created"),
  getPreviewUrl: buildActionStateRegistry("generated"),
  executeCode: buildActionStateRegistry("executed"),
  executeCommand: buildActionStateRegistry("executed"),
  deleteSandbox: buildActionStateRegistry("deleted"),
};
