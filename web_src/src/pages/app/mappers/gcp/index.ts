import type { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { cloudBuildBaseMapper, computeBaseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { CLOUD_BUILD_EXECUTION_STATE_REGISTRY } from "./cloudbuild";
import { onVMInstanceTriggerRenderer } from "./on_vm_instance";
import { onBuildCompleteTriggerRenderer } from "./on_build_complete";
import { onArtifactPushTriggerRenderer } from "./on_artifact_push";
import { onArtifactAnalysisTriggerRenderer } from "./on_artifact_analysis";
import { runTriggerMapper } from "./run_trigger";
import { invokeFunctionMapper } from "./invoke_function";
import { getArtifactMapper, getArtifactAnalysisMapper } from "./artifact_registry_mapper";
import {
  publishMessageMapper,
  createTopicMapper,
  deleteTopicMapper,
  createSubscriptionMapper,
  deleteSubscriptionMapper,
  PUBSUB_ACTION_STATE_REGISTRY,
} from "./pubsub_mapper";
import { onMessageTriggerRenderer } from "./on_message";
import { onAlertTriggerRenderer } from "./on_alert";
import { cloudDNSMapper } from "./clouddns";
import { deleteVMInstanceMapper } from "./delete_vm_instance";
import { getVMInstanceMapper } from "./get_vm_instance";
import { manageVMInstancePowerMapper, MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY } from "./manage_vm_instance_power";
import { updateVMInstanceTypeMapper } from "./update_vm_instance_type";
import { getVMInstanceMetricsMapper, GET_VM_INSTANCE_METRICS_STATE_REGISTRY } from "./get_vm_instance_metrics";
import {
  createAlertingPolicyMapper,
  getAlertingPolicyMapper,
  deleteAlertingPolicyMapper,
  updateAlertingPolicyMapper,
} from "./monitoring";
import { createSnoozeMapper } from "./create_snooze";
import { getSnoozeMapper } from "./get_snooze";
import { expireSnoozeMapper } from "./expire_snooze";
import { queryMapper, queryRangeMapper } from "./prometheus";
import { createImageMapper } from "./create_image";
import { updateImageMapper } from "./update_image";
import { deleteImageMapper } from "./delete_image";
import { createStaticIPMapper, deleteStaticIPMapper, manageStaticIPMapper } from "./static_ip";
import { createLoadBalancerMapper } from "./create_load_balancer";
import { deleteLoadBalancerMapper } from "./delete_load_balancer";
import { createFirewallRuleMapper } from "./create_firewall_rule";
import { updateFirewallRuleMapper } from "./update_firewall_rule";
import { deleteFirewallRuleMapper } from "./delete_firewall_rule";
import {
  createDatabaseMapper,
  getDatabaseMapper,
  deleteDatabaseMapper,
  createInstanceMapper,
  getInstanceMapper,
  deleteInstanceMapper,
  CLOUDSQL_CREATED_STATE_REGISTRY,
  CLOUDSQL_FETCHED_STATE_REGISTRY,
  CLOUDSQL_DELETED_STATE_REGISTRY,
} from "./cloudsql_mapper";
import {
  createBucketMapper,
  getBucketMapper,
  deleteBucketMapper,
  STORAGE_CREATED_STATE_REGISTRY,
  STORAGE_FETCHED_STATE_REGISTRY,
  STORAGE_DELETED_STATE_REGISTRY,
} from "./storage_mapper";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createVM: computeBaseMapper,
  deleteVMInstance: deleteVMInstanceMapper,
  getVMInstance: getVMInstanceMapper,
  manageVMInstancePower: manageVMInstancePowerMapper,
  updateVMInstanceType: updateVMInstanceTypeMapper,
  getVMInstanceMetrics: getVMInstanceMetricsMapper,
  createImage: createImageMapper,
  updateImage: updateImageMapper,
  deleteImage: deleteImageMapper,
  "cloudbuild.createBuild": cloudBuildBaseMapper,
  "cloudbuild.getBuild": cloudBuildBaseMapper,
  "cloudbuild.runTrigger": runTriggerMapper,
  "cloudfunctions.invokeFunction": invokeFunctionMapper,
  "artifactregistry.getArtifact": getArtifactMapper,
  "artifactregistry.getArtifactAnalysis": getArtifactAnalysisMapper,
  "pubsub.publishMessage": publishMessageMapper,
  "pubsub.createTopic": createTopicMapper,
  "pubsub.deleteTopic": deleteTopicMapper,
  "pubsub.createSubscription": createSubscriptionMapper,
  "pubsub.deleteSubscription": deleteSubscriptionMapper,
  "clouddns.createRecord": cloudDNSMapper,
  "clouddns.deleteRecord": cloudDNSMapper,
  "clouddns.updateRecord": cloudDNSMapper,
  "monitoring.createAlertingPolicy": createAlertingPolicyMapper,
  "monitoring.getAlertingPolicy": getAlertingPolicyMapper,
  "monitoring.deleteAlertingPolicy": deleteAlertingPolicyMapper,
  "monitoring.updateAlertingPolicy": updateAlertingPolicyMapper,
  "monitoring.createSnooze": createSnoozeMapper,
  "monitoring.getSnooze": getSnoozeMapper,
  "monitoring.expireSnooze": expireSnoozeMapper,
  "prometheus.query": queryMapper,
  "prometheus.queryRange": queryRangeMapper,
  "compute.createStaticIP": createStaticIPMapper,
  "compute.deleteStaticIP": deleteStaticIPMapper,
  "compute.manageStaticIP": manageStaticIPMapper,
  "compute.createLoadBalancer": createLoadBalancerMapper,
  "compute.deleteLoadBalancer": deleteLoadBalancerMapper,
  "compute.createFirewallRule": createFirewallRuleMapper,
  "compute.updateFirewallRule": updateFirewallRuleMapper,
  "compute.deleteFirewallRule": deleteFirewallRuleMapper,
  "cloudsql.createDatabase": createDatabaseMapper,
  "cloudsql.getDatabase": getDatabaseMapper,
  "cloudsql.deleteDatabase": deleteDatabaseMapper,
  "cloudsql.createInstance": createInstanceMapper,
  "cloudsql.getInstance": getInstanceMapper,
  "cloudsql.deleteInstance": deleteInstanceMapper,
  "storage.createBucket": createBucketMapper,
  "storage.getBucket": getBucketMapper,
  "storage.deleteBucket": deleteBucketMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onVMInstance: onVMInstanceTriggerRenderer,
  "cloudbuild.onBuildComplete": onBuildCompleteTriggerRenderer,
  "artifactregistry.onArtifactPush": onArtifactPushTriggerRenderer,
  "artifactregistry.onArtifactAnalysis": onArtifactAnalysisTriggerRenderer,
  "pubsub.onMessage": onMessageTriggerRenderer,
  "monitoring.onAlert": onAlertTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createVM: buildActionStateRegistry("completed"),
  deleteVMInstance: buildActionStateRegistry("completed"),
  getVMInstance: buildActionStateRegistry("completed"),
  manageVMInstancePower: MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY,
  updateVMInstanceType: buildActionStateRegistry("completed"),
  getVMInstanceMetrics: GET_VM_INSTANCE_METRICS_STATE_REGISTRY,
  createImage: buildActionStateRegistry("created"),
  updateImage: buildActionStateRegistry("updated"),
  deleteImage: buildActionStateRegistry("deleted"),
  "cloudbuild.createBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.getBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.runTrigger": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudfunctions.invokeFunction": buildActionStateRegistry("completed"),
  "artifactregistry.getArtifact": buildActionStateRegistry("completed"),
  "artifactregistry.getArtifactAnalysis": buildActionStateRegistry("completed"),
  "pubsub.publishMessage": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.createTopic": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.deleteTopic": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.createSubscription": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.deleteSubscription": PUBSUB_ACTION_STATE_REGISTRY,
  "clouddns.createRecord": buildActionStateRegistry("completed"),
  "clouddns.deleteRecord": buildActionStateRegistry("completed"),
  "clouddns.updateRecord": buildActionStateRegistry("completed"),
  "monitoring.createAlertingPolicy": buildActionStateRegistry("completed"),
  "monitoring.getAlertingPolicy": buildActionStateRegistry("completed"),
  "monitoring.deleteAlertingPolicy": buildActionStateRegistry("completed"),
  "monitoring.updateAlertingPolicy": buildActionStateRegistry("completed"),
  "monitoring.createSnooze": buildActionStateRegistry("created"),
  "monitoring.getSnooze": buildActionStateRegistry("fetched"),
  "monitoring.expireSnooze": buildActionStateRegistry("expired"),
  "prometheus.query": buildActionStateRegistry("completed"),
  "prometheus.queryRange": buildActionStateRegistry("completed"),
  "compute.createStaticIP": buildActionStateRegistry("completed"),
  "compute.deleteStaticIP": buildActionStateRegistry("completed"),
  "compute.manageStaticIP": buildActionStateRegistry("completed"),
  "compute.createLoadBalancer": buildActionStateRegistry("created"),
  "compute.deleteLoadBalancer": buildActionStateRegistry("deleted"),
  "compute.createFirewallRule": buildActionStateRegistry("created"),
  "compute.updateFirewallRule": buildActionStateRegistry("updated"),
  "compute.deleteFirewallRule": buildActionStateRegistry("deleted"),
  "cloudsql.createDatabase": CLOUDSQL_CREATED_STATE_REGISTRY,
  "cloudsql.getDatabase": CLOUDSQL_FETCHED_STATE_REGISTRY,
  "cloudsql.deleteDatabase": CLOUDSQL_DELETED_STATE_REGISTRY,
  "cloudsql.createInstance": CLOUDSQL_CREATED_STATE_REGISTRY,
  "cloudsql.getInstance": CLOUDSQL_FETCHED_STATE_REGISTRY,
  "cloudsql.deleteInstance": CLOUDSQL_DELETED_STATE_REGISTRY,
  "storage.createBucket": STORAGE_CREATED_STATE_REGISTRY,
  "storage.getBucket": STORAGE_FETCHED_STATE_REGISTRY,
  "storage.deleteBucket": STORAGE_DELETED_STATE_REGISTRY,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};
