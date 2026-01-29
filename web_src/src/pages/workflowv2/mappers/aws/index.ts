import { ComponentBaseMapper, TriggerRenderer } from "../types";
import { runFunctionMapper } from "./lambda/run_function";
import { onImagePushTriggerRenderer } from "./ecr/on_image_push";
import { onImageScanTriggerRenderer } from "./ecr/on_image_scan";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  "lambda.runFunction": runFunctionMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  "ecr.onImagePush": onImagePushTriggerRenderer,
  "ecr.onImageScan": onImageScanTriggerRenderer,
};
