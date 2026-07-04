import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { runFunctionMapper } from "./lambda/run_function";
import { onImagePushTriggerRenderer } from "./ecr/on_image_push";
import { onImageScanTriggerRenderer } from "./ecr/on_image_scan";
import { getImageMapper } from "./ecr/get_image";
import { getImageScanFindingsMapper } from "./ecr/get_image_scan_findings";
import { buildActionStateRegistry } from "../utils";
import { scanImageMapper } from "./ecr/scan_image";
import { onPackageVersionTriggerRenderer } from "./codeartifact/on_package_version";
import { getPackageVersionMapper } from "./codeartifact/get_package_version";
import { createQueueMapper, deleteQueueMapper, getQueueMapper, purgeQueueMapper, sendMessageMapper } from "./sqs";
import { createRepositoryMapper } from "./codeartifact/create_repository";
import { copyPackageVersionsMapper } from "./codeartifact/copy_package_versions";
import { deletePackageVersionsMapper } from "./codeartifact/delete_package_versions";
import { deleteRepositoryMapper } from "./codeartifact/delete_repository";
import { disposePackageVersionsMapper } from "./codeartifact/dispose_package_versions";
import { updatePackageVersionsStatusMapper } from "./codeartifact/update_package_versions_status";
import { onAlarmTriggerRenderer } from "./cloudwatch/on_alarm";
import { createServiceMapper } from "./ecs/create_service";
import { createRecordMapper } from "./route53/create_record";
import { upsertRecordMapper } from "./route53/upsert_record";
import { deleteRecordMapper } from "./route53/delete_record";
import { describeServiceMapper } from "./ecs/describe_service";
import { executeCommandMapper } from "./ecs/execute_command";
import { runTaskMapper } from "./ecs/run_task";
import { stopTaskMapper } from "./ecs/stop_task";
import { updateServiceMapper } from "./ecs/update_service";
import { onTopicMessageTriggerRenderer } from "./sns/on_topic_message";
import { createTopicMapper } from "./sns/create_topic";
import { deleteTopicMapper } from "./sns/delete_topic";
import { getSubscriptionMapper } from "./sns/get_subscription";
import { getTopicMapper } from "./sns/get_topic";
import { publishMessageMapper } from "./sns/publish_message";
import { getPipelineExecutionMapper } from "./codepipeline/get_pipeline_execution";
import { retryStageExecutionMapper } from "./codepipeline/retry_stage_execution";
import { RUN_PIPELINE_STATE_REGISTRY, runPipelineMapper } from "./codepipeline/run_pipeline";
import { getPipelineMapper } from "./codepipeline/get_pipeline";
import { onPipelineTriggerRenderer } from "./codepipeline/on_pipeline";
import { onImageTriggerRenderer } from "./ec2/on_image";
import { onEc2AlarmTriggerRenderer } from "./ec2/on_alarm";
import { createAlarmMapper } from "./ec2/create_alarm";
import { createImageMapper } from "./ec2/create_image";
import { CREATE_INSTANCE_STATE_REGISTRY, createInstanceMapper } from "./ec2/create_instance";
import { deleteInstanceMapper } from "./ec2/delete_instance";
import { getAlarmMapper } from "./ec2/get_alarm";
import { getImageMapper as getEc2ImageMapper } from "./ec2/get_image";
import { getInstanceMapper } from "./ec2/get_instance";
import { getInstanceMetricsMapper } from "./ec2/get_instance_metrics";
import { allocateElasticIPMapper } from "./ec2/allocate_elastic_ip";
import { manageElasticIPMapper, MANAGE_ELASTIC_IP_STATE_REGISTRY } from "./ec2/manage_elastic_ip";
import { manageInstancePowerMapper, MANAGE_INSTANCE_POWER_STATE_REGISTRY } from "./ec2/manage_instance_power";
import { releaseElasticIPMapper } from "./ec2/release_elastic_ip";
import { updateInstanceMapper } from "./ec2/update_instance";
import { copyImageMapper } from "./ec2/copy_image";
import { deregisterImageMapper } from "./ec2/deregister_image";
import { enableImageMapper } from "./ec2/enable_image";
import { disableImageMapper } from "./ec2/disable_image";
import { enableImageDeprecationMapper } from "./ec2/enable_image_deprecation";
import { disableImageDeprecationMapper } from "./ec2/disable_image_deprecation";
import {
  createWorkspaceMapper,
  deleteWorkspaceMapper,
  getWorkspaceMapper,
  queryMapper,
  queryRangeMapper,
  updateWorkspaceMapper,
} from "./prometheus";
import { createLoadBalancerMapper } from "./ec2/create_load_balancer";
import { deleteLoadBalancerMapper } from "./ec2/delete_load_balancer";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  "codepipeline.getPipeline": getPipelineMapper,
  "codepipeline.getPipelineExecution": getPipelineExecutionMapper,
  "codepipeline.retryStageExecution": retryStageExecutionMapper,
  "codepipeline.runPipeline": runPipelineMapper,
  "lambda.runFunction": runFunctionMapper,
  "ecs.createService": createServiceMapper,
  "ecs.describeService": describeServiceMapper,
  "ecs.executeCommand": executeCommandMapper,
  "ecs.runTask": runTaskMapper,
  "ecs.stopTask": stopTaskMapper,
  "ecs.updateService": updateServiceMapper,
  "ecr.getImage": getImageMapper,
  "ecr.getImageScanFindings": getImageScanFindingsMapper,
  "ecr.scanImage": scanImageMapper,
  "prometheus.createWorkspace": createWorkspaceMapper,
  "prometheus.getWorkspace": getWorkspaceMapper,
  "prometheus.updateWorkspace": updateWorkspaceMapper,
  "prometheus.deleteWorkspace": deleteWorkspaceMapper,
  "prometheus.query": queryMapper,
  "prometheus.queryRange": queryRangeMapper,
  "codeArtifact.copyPackageVersions": copyPackageVersionsMapper,
  "codeArtifact.createRepository": createRepositoryMapper,
  "codeArtifact.deletePackageVersions": deletePackageVersionsMapper,
  "codeArtifact.deleteRepository": deleteRepositoryMapper,
  "codeArtifact.disposePackageVersions": disposePackageVersionsMapper,
  "codeArtifact.getPackageVersion": getPackageVersionMapper,
  "sqs.createQueue": createQueueMapper,
  "sqs.getQueue": getQueueMapper,
  "sqs.sendMessage": sendMessageMapper,
  "sqs.deleteQueue": deleteQueueMapper,
  "sqs.purgeQueue": purgeQueueMapper,
  "codeArtifact.updatePackageVersionsStatus": updatePackageVersionsStatusMapper,
  "route53.createRecord": createRecordMapper,
  "route53.upsertRecord": upsertRecordMapper,
  "route53.deleteRecord": deleteRecordMapper,
  "sns.getTopic": getTopicMapper,
  "sns.getSubscription": getSubscriptionMapper,
  "sns.createTopic": createTopicMapper,
  "sns.deleteTopic": deleteTopicMapper,
  "sns.publishMessage": publishMessageMapper,
  "ec2.allocateElasticIP": allocateElasticIPMapper,
  "ec2.manageElasticIP": manageElasticIPMapper,
  "ec2.copyImage": copyImageMapper,
  "ec2.createAlarm": createAlarmMapper,
  "ec2.createImage": createImageMapper,
  "ec2.createInstance": createInstanceMapper,
  "ec2.deregisterImage": deregisterImageMapper,
  "ec2.deleteInstance": deleteInstanceMapper,
  "ec2.disableImage": disableImageMapper,
  "ec2.disableImageDeprecation": disableImageDeprecationMapper,
  "ec2.enableImage": enableImageMapper,
  "ec2.enableImageDeprecation": enableImageDeprecationMapper,
  "ec2.getAlarm": getAlarmMapper,
  "ec2.getImage": getEc2ImageMapper,
  "ec2.getInstance": getInstanceMapper,
  "ec2.getInstanceMetrics": getInstanceMetricsMapper,
  "ec2.manageInstancePower": manageInstancePowerMapper,
  "ec2.releaseElasticIP": releaseElasticIPMapper,
  "ec2.updateInstance": updateInstanceMapper,
  "ec2.createLoadBalancer": createLoadBalancerMapper,
  "ec2.deleteLoadBalancer": deleteLoadBalancerMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  "cloudwatch.onAlarm": onAlarmTriggerRenderer,
  "codeArtifact.onPackageVersion": onPackageVersionTriggerRenderer,
  "codepipeline.onPipeline": onPipelineTriggerRenderer,
  "ecr.onImagePush": onImagePushTriggerRenderer,
  "ecr.onImageScan": onImageScanTriggerRenderer,
  "sns.onTopicMessage": onTopicMessageTriggerRenderer,
  "ec2.onAlarm": onEc2AlarmTriggerRenderer,
  "ec2.onImage": onImageTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  "codepipeline.getPipeline": buildActionStateRegistry("retrieved"),
  "codepipeline.getPipelineExecution": buildActionStateRegistry("retrieved"),
  "codepipeline.retryStageExecution": buildActionStateRegistry("retried"),
  "codepipeline.runPipeline": RUN_PIPELINE_STATE_REGISTRY,
  "ecs.createService": buildActionStateRegistry("created"),
  "ecs.describeService": buildActionStateRegistry("described"),
  "ecs.executeCommand": buildActionStateRegistry("executed"),
  "ecs.runTask": buildActionStateRegistry("started"),
  "ecs.stopTask": buildActionStateRegistry("stopped"),
  "ecs.updateService": buildActionStateRegistry("updated"),
  "ecr.getImage": buildActionStateRegistry("retrieved"),
  "ecr.getImageScanFindings": buildActionStateRegistry("retrieved"),
  "ecr.scanImage": buildActionStateRegistry("scanned"),
  "prometheus.createWorkspace": buildActionStateRegistry("created"),
  "prometheus.getWorkspace": buildActionStateRegistry("retrieved"),
  "prometheus.updateWorkspace": buildActionStateRegistry("updated"),
  "prometheus.deleteWorkspace": buildActionStateRegistry("deleted"),
  "prometheus.query": buildActionStateRegistry("success"),
  "prometheus.queryRange": buildActionStateRegistry("success"),
  "codeArtifact.copyPackageVersions": buildActionStateRegistry("copied"),
  "codeArtifact.createRepository": buildActionStateRegistry("created"),
  "codeArtifact.deletePackageVersions": buildActionStateRegistry("deleted"),
  "codeArtifact.deleteRepository": buildActionStateRegistry("deleted"),
  "codeArtifact.disposePackageVersions": buildActionStateRegistry("disposed"),
  "codeArtifact.getPackageVersion": buildActionStateRegistry("retrieved"),
  "sqs.createQueue": buildActionStateRegistry("created"),
  "sqs.getQueue": buildActionStateRegistry("retrieved"),
  "sqs.sendMessage": buildActionStateRegistry("sent"),
  "sqs.deleteQueue": buildActionStateRegistry("deleted"),
  "sqs.purgeQueue": buildActionStateRegistry("purged"),
  "codeArtifact.updatePackageVersionsStatus": buildActionStateRegistry("updated"),
  "route53.createRecord": buildActionStateRegistry("created"),
  "route53.upsertRecord": buildActionStateRegistry("upserted"),
  "route53.deleteRecord": buildActionStateRegistry("deleted"),
  "sns.getTopic": buildActionStateRegistry("retrieved"),
  "sns.getSubscription": buildActionStateRegistry("retrieved"),
  "sns.createTopic": buildActionStateRegistry("created"),
  "sns.deleteTopic": buildActionStateRegistry("deleted"),
  "sns.publishMessage": buildActionStateRegistry("published"),
  "ec2.allocateElasticIP": buildActionStateRegistry("allocated"),
  "ec2.manageElasticIP": MANAGE_ELASTIC_IP_STATE_REGISTRY,
  "ec2.copyImage": buildActionStateRegistry("copied"),
  "ec2.createAlarm": buildActionStateRegistry("created"),
  "ec2.createImage": buildActionStateRegistry("created"),
  "ec2.createInstance": CREATE_INSTANCE_STATE_REGISTRY,
  "ec2.deregisterImage": buildActionStateRegistry("deregistered"),
  "ec2.deleteInstance": buildActionStateRegistry("deleted"),
  "ec2.disableImage": buildActionStateRegistry("disabled"),
  "ec2.disableImageDeprecation": buildActionStateRegistry("disabled"),
  "ec2.enableImage": buildActionStateRegistry("enabled"),
  "ec2.enableImageDeprecation": buildActionStateRegistry("enabled"),
  "ec2.getAlarm": buildActionStateRegistry("retrieved"),
  "ec2.getImage": buildActionStateRegistry("retrieved"),
  "ec2.getInstance": buildActionStateRegistry("retrieved"),
  "ec2.getInstanceMetrics": buildActionStateRegistry("retrieved"),
  "ec2.manageInstancePower": MANAGE_INSTANCE_POWER_STATE_REGISTRY,
  "ec2.releaseElasticIP": buildActionStateRegistry("released"),
  "ec2.updateInstance": buildActionStateRegistry("updated"),
  "ec2.createLoadBalancer": buildActionStateRegistry("created"),
  "ec2.deleteLoadBalancer": buildActionStateRegistry("deleted"),
};
