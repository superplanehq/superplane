import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createSandbox: baseMapper,
  executeCode: baseMapper,
  executeCommand: baseMapper,
  deleteSandbox: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createSandbox: buildActionStateRegistry("created"),
  executeCode: buildActionStateRegistry("executed"),
  executeCommand: buildActionStateRegistry("executed"),
  deleteSandbox: buildActionStateRegistry("deleted"),
};
