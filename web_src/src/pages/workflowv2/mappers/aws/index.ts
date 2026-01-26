import { ComponentBaseMapper, TriggerRenderer } from "../types";
import { runFunctionMapper } from "./lambda/run_function";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  "lambda.runFunction": runFunctionMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};
