import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { runFunctionMapper } from "./lambda/run_function";
import { onImagePushTriggerRenderer } from "./ecr/on_image_push";
import { onImageScanTriggerRenderer } from "./ecr/on_image_scan";
import { getImageMapper } from "./ecr/get_image";
import { getImageScanFindingsMapper } from "./ecr/get_image_scan_findings";
import { buildActionStateRegistry } from "../utils";
import { scanImageMapper } from "./ecr/scan_image";
import { onPackageVersionTriggerRenderer } from "./codeartifact/on_package_version";
import { getPackageVersionMapper } from "./codeartifact/get_package_version";
import { onTopicMessageTriggerRenderer } from "./sns/on_topic_message";
import {
  createTopicMapper,
  deleteTopicMapper,
  getSubscriptionMapper,
  getTopicMapper,
  publishMessageMapper,
  subscribeMapper,
  unsubscribeMapper,
} from "./sns/components";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  "lambda.runFunction": runFunctionMapper,
  "ecr.getImage": getImageMapper,
  "ecr.getImageScanFindings": getImageScanFindingsMapper,
  "ecr.scanImage": scanImageMapper,
  "codeArtifact.getPackageVersion": getPackageVersionMapper,
  "sns.getTopic": getTopicMapper,
  "sns.getSubscription": getSubscriptionMapper,
  "sns.createTopic": createTopicMapper,
  "sns.deleteTopic": deleteTopicMapper,
  "sns.publishMessage": publishMessageMapper,
  "sns.subscribe": subscribeMapper,
  "sns.unsubscribe": unsubscribeMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  "codeArtifact.onPackageVersion": onPackageVersionTriggerRenderer,
  "ecr.onImagePush": onImagePushTriggerRenderer,
  "ecr.onImageScan": onImageScanTriggerRenderer,
  "sns.onTopicMessage": onTopicMessageTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  "ecr.getImage": buildActionStateRegistry("retrieved"),
  "ecr.getImageScanFindings": buildActionStateRegistry("retrieved"),
  "ecr.scanImage": buildActionStateRegistry("scanned"),
  "codeArtifact.getPackageVersion": buildActionStateRegistry("retrieved"),
  "sns.getTopic": buildActionStateRegistry("retrieved"),
  "sns.getSubscription": buildActionStateRegistry("retrieved"),
  "sns.createTopic": buildActionStateRegistry("created"),
  "sns.deleteTopic": buildActionStateRegistry("deleted"),
  "sns.publishMessage": buildActionStateRegistry("published"),
  "sns.subscribe": buildActionStateRegistry("subscribed"),
  "sns.unsubscribe": buildActionStateRegistry("unsubscribed"),
};
