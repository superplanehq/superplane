import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { runFunctionMapper } from "./lambda/run_function";
import { onImagePushTriggerRenderer } from "./ecr/on_image_push";
import { onImageScanTriggerRenderer } from "./ecr/on_image_scan";
import { getImageMapper } from "./ecr/get_image";
import { getImageScanFindingsMapper } from "./ecr/get_image_scan_findings";
import { buildActionStateRegistry } from "../utils";
import { scanImageMapper } from "./ecr/scan_image";
import { onPackageVersionTriggerRenderer } from "./codeartifact/on_package_version";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  "lambda.runFunction": runFunctionMapper,
  "ecr.getImage": getImageMapper,
  "ecr.getImageScanFindings": getImageScanFindingsMapper,
  "ecr.scanImage": scanImageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  "codeArtifact.onPackageVersion": onPackageVersionTriggerRenderer,
  "ecr.onImagePush": onImagePushTriggerRenderer,
  "ecr.onImageScan": onImageScanTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  "ecr.getImage": buildActionStateRegistry("retrieved"),
  "ecr.getImageScanFindings": buildActionStateRegistry("retrieved"),
  "ecr.scanImage": buildActionStateRegistry("scanned"),
};
