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
import { createRepositoryMapper } from "./codeartifact/create_repository";
import { copyPackageVersionsMapper } from "./codeartifact/copy_package_versions";
import { deletePackageVersionsMapper } from "./codeartifact/delete_package_versions";
import { deleteRepositoryMapper } from "./codeartifact/delete_repository";
import { disposePackageVersionsMapper } from "./codeartifact/dispose_package_versions";
import { updatePackageVersionsStatusMapper } from "./codeartifact/update_package_versions_status";
import { onAlarmTriggerRenderer } from "./cloudwatch/on_alarm";
import { createRecordMapper } from "./route53/create_record";
import { upsertRecordMapper } from "./route53/upsert_record";
import { deleteRecordMapper } from "./route53/delete_record";
import { describeServiceMapper } from "./ecs/describe_service";
import { runTaskMapper } from "./ecs/run_task";
import { stopTaskMapper } from "./ecs/stop_task";
import { onTopicMessageTriggerRenderer } from "./sns/on_topic_message";
import { createTopicMapper } from "./sns/create_topic";
import { deleteTopicMapper } from "./sns/delete_topic";
import { getSubscriptionMapper } from "./sns/get_subscription";
import { getTopicMapper } from "./sns/get_topic";
import { publishMessageMapper } from "./sns/publish_message";
import { onImageTriggerRenderer } from "./ec2/on_image";
import { createImageMapper } from "./ec2/create_image";
import { getImageMapper as getEc2ImageMapper } from "./ec2/get_image";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  "lambda.runFunction": runFunctionMapper,
  "ecs.describeService": describeServiceMapper,
  "ecs.runTask": runTaskMapper,
  "ecs.stopTask": stopTaskMapper,
  "ecr.getImage": getImageMapper,
  "ecr.getImageScanFindings": getImageScanFindingsMapper,
  "ecr.scanImage": scanImageMapper,
  "codeArtifact.copyPackageVersions": copyPackageVersionsMapper,
  "codeArtifact.createRepository": createRepositoryMapper,
  "codeArtifact.deletePackageVersions": deletePackageVersionsMapper,
  "codeArtifact.deleteRepository": deleteRepositoryMapper,
  "codeArtifact.disposePackageVersions": disposePackageVersionsMapper,
  "codeArtifact.getPackageVersion": getPackageVersionMapper,
  "codeArtifact.updatePackageVersionsStatus": updatePackageVersionsStatusMapper,
  "route53.createRecord": createRecordMapper,
  "route53.upsertRecord": upsertRecordMapper,
  "route53.deleteRecord": deleteRecordMapper,
  "sns.getTopic": getTopicMapper,
  "sns.getSubscription": getSubscriptionMapper,
  "sns.createTopic": createTopicMapper,
  "sns.deleteTopic": deleteTopicMapper,
  "sns.publishMessage": publishMessageMapper,
  "ec2.createImage": createImageMapper,
  "ec2.getImage": getEc2ImageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  "cloudwatch.onAlarm": onAlarmTriggerRenderer,
  "codeArtifact.onPackageVersion": onPackageVersionTriggerRenderer,
  "ecr.onImagePush": onImagePushTriggerRenderer,
  "ecr.onImageScan": onImageScanTriggerRenderer,
  "sns.onTopicMessage": onTopicMessageTriggerRenderer,
  "ec2.onImage": onImageTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  "ecs.describeService": buildActionStateRegistry("described"),
  "ecs.runTask": buildActionStateRegistry("started"),
  "ecs.stopTask": buildActionStateRegistry("stopped"),
  "ecr.getImage": buildActionStateRegistry("retrieved"),
  "ecr.getImageScanFindings": buildActionStateRegistry("retrieved"),
  "ecr.scanImage": buildActionStateRegistry("scanned"),
  "codeArtifact.copyPackageVersions": buildActionStateRegistry("copied"),
  "codeArtifact.createRepository": buildActionStateRegistry("created"),
  "codeArtifact.deletePackageVersions": buildActionStateRegistry("deleted"),
  "codeArtifact.deleteRepository": buildActionStateRegistry("deleted"),
  "codeArtifact.disposePackageVersions": buildActionStateRegistry("disposed"),
  "codeArtifact.getPackageVersion": buildActionStateRegistry("retrieved"),
  "codeArtifact.updatePackageVersionsStatus": buildActionStateRegistry("updated"),
  "route53.createRecord": buildActionStateRegistry("created"),
  "route53.upsertRecord": buildActionStateRegistry("upserted"),
  "route53.deleteRecord": buildActionStateRegistry("deleted"),
  "sns.getTopic": buildActionStateRegistry("retrieved"),
  "sns.getSubscription": buildActionStateRegistry("retrieved"),
  "sns.createTopic": buildActionStateRegistry("created"),
  "sns.deleteTopic": buildActionStateRegistry("deleted"),
  "sns.publishMessage": buildActionStateRegistry("published"),
  "ec2.createImage": buildActionStateRegistry("created"),
  "ec2.getImage": buildActionStateRegistry("retrieved"),
};
