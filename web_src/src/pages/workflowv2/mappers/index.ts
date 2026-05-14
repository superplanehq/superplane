import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { addMemoryMapper } from "./addMemory";
import {
  componentMappers as awsComponentMappers,
  eventStateRegistry as awsEventStateRegistry,
  triggerRenderers as awsTriggerRenderers,
} from "./aws";
import {
  componentMappers as azureComponentMappers,
  eventStateRegistry as azureEventStateRegistry,
  triggerRenderers as azureTriggerRenderers,
} from "./azure/index";
import { triggerRenderers as bitbucketTriggerRenderers } from "./bitbucket/index";
import {
  componentMappers as circleCIComponentMappers,
  eventStateRegistry as circleCIEventStateRegistry,
  triggerRenderers as circleCITriggerRenderers,
} from "./circleci/index";
import {
  componentMappers as claudeComponentMappers,
  eventStateRegistry as claudeEventStateRegistry,
  triggerRenderers as claudeTriggerRenderers,
} from "./claude/index";
import {
  componentMappers as cloudflareComponentMappers,
  eventStateRegistry as cloudflareEventStateRegistry,
  triggerRenderers as cloudflareTriggerRenderers,
} from "./cloudflare/index";
import {
  componentMappers as cursorComponentMappers,
  eventStateRegistry as cursorEventStateRegistry,
  triggerRenderers as cursorTriggerRenderers,
} from "./cursor/index";
import {
  componentMappers as dash0ComponentMappers,
  eventStateRegistry as dash0EventStateRegistry,
  triggerRenderers as dash0TriggerRenderers,
} from "./dash0/index";
import {
  componentMappers as datadogComponentMappers,
  eventStateRegistry as datadogEventStateRegistry,
  triggerRenderers as datadogTriggerRenderers,
} from "./datadog/index";
import {
  componentMappers as daytonaComponentMappers,
  eventStateRegistry as daytonaEventStateRegistry,
  triggerRenderers as daytonaTriggerRenderers,
} from "./daytona/index";
import { defaultTriggerRenderer } from "./default";
import { deleteMemoryMapper } from "./deleteMemory";
import {
  componentMappers as digitaloceanComponentMappers,
  eventStateRegistry as digitaloceanEventStateRegistry,
  triggerRenderers as digitaloceanTriggerRenderers,
} from "./digitalocean/index";
import {
  componentMappers as discordComponentMappers,
  eventStateRegistry as discordEventStateRegistry,
  triggerRenderers as discordTriggerRenderers,
} from "./discord";
import {
  componentMappers as dockerhubComponentMappers,
  customFieldRenderers as dockerhubCustomFieldRenderers,
  eventStateRegistry as dockerhubEventStateRegistry,
  triggerRenderers as dockerhubTriggerRenderers,
} from "./dockerhub";
import {
  componentMappers as firehydrantComponentMappers,
  eventStateRegistry as firehydrantEventStateRegistry,
  triggerRenderers as firehydrantTriggerRenderers,
} from "./firehydrant/index";
import {
  componentMappers as githubComponentMappers,
  eventStateRegistry as githubEventStateRegistry,
  triggerRenderers as githubTriggerRenderers,
} from "./github/index";
import {
  componentMappers as gitlabComponentMappers,
  eventStateRegistry as gitlabEventStateRegistry,
  triggerRenderers as gitlabTriggerRenderers,
} from "./gitlab/index";
import {
  componentMappers as grafanaComponentMappers,
  customFieldRenderers as grafanaCustomFieldRenderers,
  eventStateRegistry as grafanaEventStateRegistry,
  triggerRenderers as grafanaTriggerRenderers,
} from "./grafana/index";
import { GRAPHQL_STATE_REGISTRY, graphqlMapper } from "./graphql";
import {
  componentMappers as harnessComponentMappers,
  eventStateRegistry as harnessEventStateRegistry,
  triggerRenderers as harnessTriggerRenderers,
} from "./harness";
import { componentMappers as hetznerComponentMappers } from "./hetzner/index";
import { HTTP_STATE_REGISTRY, httpMapper } from "./http";
import { IF_STATE_REGISTRY, ifMapper } from "./if";
import {
  componentMappers as incidentComponentMappers,
  customFieldRenderers as incidentCustomFieldRenderers,
  eventStateRegistry as incidentEventStateRegistry,
  triggerRenderers as incidentTriggerRenderers,
} from "./incident/index";
import {
  componentMappers as jfrogArtifactoryComponentMappers,
  eventStateRegistry as jfrogArtifactoryEventStateRegistry,
  triggerRenderers as jfrogArtifactoryTriggerRenderers,
} from "./jfrogArtifactory/index";
import {
  componentMappers as launchdarklyComponentMappers,
  eventStateRegistry as launchdarklyEventStateRegistry,
  triggerRenderers as launchdarklyTriggerRenderers,
} from "./launchdarkly/index";
import {
  componentMappers as logfireComponentMappers,
  eventStateRegistry as logfireEventStateRegistry,
  triggerRenderers as logfireTriggerRenderers,
} from "./logfire/index";
import {
  componentMappers as newrelicComponentMappers,
  customFieldRenderers as newrelicCustomFieldRenderers,
  eventStateRegistry as newrelicEventStateRegistry,
  triggerRenderers as newrelicTriggerRenderers,
} from "./newrelic/index";
import { noopMapper } from "./noop";
import {
  componentMappers as octopusComponentMappers,
  eventStateRegistry as octopusEventStateRegistry,
  triggerRenderers as octopusTriggerRenderers,
} from "./octopus/index";
import {
  componentMappers as openaiComponentMappers,
  eventStateRegistry as openaiEventStateRegistry,
  triggerRenderers as openaiTriggerRenderers,
} from "./openai/index";
import {
  componentMappers as pagerdutyComponentMappers,
  eventStateRegistry as pagerdutyEventStateRegistry,
  triggerRenderers as pagerdutyTriggerRenderers,
} from "./pagerduty/index";
import {
  componentMappers as perplexityComponentMappers,
  eventStateRegistry as perplexityEventStateRegistry,
  triggerRenderers as perplexityTriggerRenderers,
} from "./perplexity/index";
import {
  componentMappers as prometheusComponentMappers,
  customFieldRenderers as prometheusCustomFieldRenderers,
  eventStateRegistry as prometheusEventStateRegistry,
  triggerRenderers as prometheusTriggerRenderers,
} from "./prometheus/index";
import { readMemoryMapper } from "./readMemory";
import {
  componentMappers as renderComponentMappers,
  eventStateRegistry as renderEventStateRegistry,
  triggerRenderers as renderTriggerRenderers,
} from "./render";
import {
  componentMappers as rootlyComponentMappers,
  eventStateRegistry as rootlyEventStateRegistry,
  triggerRenderers as rootlyTriggerRenderers,
} from "./rootly/index";
import { scheduleCustomFieldRenderer, scheduleTriggerRenderer } from "./schedule";
import {
  componentMappers as semaphoreComponentMappers,
  eventStateRegistry as semaphoreEventStateRegistry,
  triggerRenderers as semaphoreTriggerRenderers,
} from "./semaphore/index";
import {
  componentMappers as sendgridComponentMappers,
  eventStateRegistry as sendgridEventStateRegistry,
  triggerRenderers as sendgridTriggerRenderers,
} from "./sendgrid";
import {
  componentMappers as sentryComponentMappers,
  eventStateRegistry as sentryEventStateRegistry,
  triggerRenderers as sentryTriggerRenderers,
} from "./sentry/index";
import {
  componentMappers as slackComponentMappers,
  eventStateRegistry as slackEventStateRegistry,
  triggerRenderers as slackTriggerRenderers,
} from "./slack";
import {
  componentMappers as smtpComponentMappers,
  eventStateRegistry as smtpEventStateRegistry,
  triggerRenderers as smtpTriggerRenderers,
} from "./smtp";
import {
  componentMappers as statuspageComponentMappers,
  eventStateRegistry as statuspageEventStateRegistry,
  triggerRenderers as statuspageTriggerRenderers,
} from "./statuspage";
import {
  componentMappers as teamsComponentMappers,
  eventStateRegistry as teamsEventStateRegistry,
  triggerRenderers as teamsTriggerRenderers,
} from "./teams";
import {
  componentMappers as telegramComponentMappers,
  eventStateRegistry as telegramEventStateRegistry,
  triggerRenderers as telegramTriggerRenderers,
} from "./telegram";
import { TIME_GATE_STATE_REGISTRY, timeGateMapper } from "./timegate";
import type {
  ComponentBaseMapper,
  CustomFieldRenderer,
  EventStateRegistry,
  TriggerEventContext,
  TriggerRenderer,
  TriggerRendererContext,
} from "./types";
import { updateMemoryMapper } from "./updateMemory";
import { upsertMemoryMapper } from "./upsertMemory";
import { webhookCustomFieldRenderer, webhookTriggerRenderer } from "./webhook";

import {
  componentMappers as elasticComponentMappers,
  eventStateRegistry as elasticEventStateRegistry,
  triggerRenderers as elasticTriggerRenderers,
} from "./elastic/index";
import {
  componentMappers as gcpComponentMappers,
  customFieldRenderers as gcpCustomFieldRenderers,
  eventStateRegistry as gcpEventStateRegistry,
  triggerRenderers as gcpTriggerRenderers,
} from "./gcp";
import {
  componentMappers as honeycombComponentMappers,
  eventStateRegistry as honeycombEventStateRegistry,
  triggerRenderers as honeycombTriggerRenderers,
} from "./honeycomb/index";
import {
  componentMappers as ociComponentMappers,
  eventStateRegistry as ociEventStateRegistry,
  triggerRenderers as ociTriggerRenderers,
} from "./oci/index";
import {
  componentMappers as servicenowComponentMappers,
  customFieldRenderers as servicenowCustomFieldRenderers,
  eventStateRegistry as servicenowEventStateRegistry,
  triggerRenderers as servicenowTriggerRenderers,
} from "./servicenow/index";

import { buildExecutionInfo, buildNodeInfo } from "../utils";
import { APPROVAL_STATE_REGISTRY, approvalMapper } from "./approval";
import { FILTER_STATE_REGISTRY, filterMapper } from "./filter";
import { MERGE_STATE_REGISTRY, mergeMapper } from "./merge";
import { RUNNER_STATE_REGISTRY, runnerMapper } from "./runner";
import { createSafeComponentMapper, createSafeCustomFieldRenderer, createSafeTriggerRenderer } from "./safeMappers";
import { SEND_EMAIL_STATE_REGISTRY, sendEmailMapper } from "./sendEmail";
import { SSH_STATE_REGISTRY, sshMapper } from "./ssh";
import { startTriggerRenderer } from "./start";
import { DEFAULT_STATE_REGISTRY } from "./stateRegistry";
import { WAIT_STATE_REGISTRY, waitCustomFieldRenderer, waitMapper } from "./wait";

/**
 * Registry mapping trigger names to their renderers.
 * Any trigger type not in this registry will use the defaultTriggerRenderer.
 */
const triggerRenderers: Record<string, TriggerRenderer> = {
  schedule: scheduleTriggerRenderer,
  webhook: webhookTriggerRenderer,
  start: startTriggerRenderer,
};

const componentBaseMappers: Record<string, ComponentBaseMapper> = {
  noop: noopMapper,
  addMemory: addMemoryMapper,
  deleteMemory: deleteMemoryMapper,
  readMemory: readMemoryMapper,
  updateMemory: updateMemoryMapper,
  upsertMemory: upsertMemoryMapper,
  if: ifMapper,
  http: httpMapper,
  graphql: graphqlMapper,
  ssh: sshMapper,
  runner: runnerMapper,
  timeGate: timeGateMapper,
  filter: filterMapper,
  wait: waitMapper,
  approval: approvalMapper,
  merge: mergeMapper,
  sendEmail: sendEmailMapper,
};

const appMappers: Record<string, Record<string, ComponentBaseMapper>> = {
  cloudflare: cloudflareComponentMappers,
  digitalocean: digitaloceanComponentMappers,
  semaphore: semaphoreComponentMappers,
  github: githubComponentMappers,
  gitlab: gitlabComponentMappers,
  grafana: grafanaComponentMappers,
  pagerduty: pagerdutyComponentMappers,
  dash0: dash0ComponentMappers,
  daytona: daytonaComponentMappers,
  datadog: datadogComponentMappers,
  slack: slackComponentMappers,
  smtp: smtpComponentMappers,
  sendgrid: sendgridComponentMappers,
  sentry: sentryComponentMappers,
  render: renderComponentMappers,
  rootly: rootlyComponentMappers,
  incident: incidentComponentMappers,
  newrelic: newrelicComponentMappers,
  firehydrant: firehydrantComponentMappers,
  launchdarkly: launchdarklyComponentMappers,
  aws: awsComponentMappers,
  azure: azureComponentMappers,
  discord: discordComponentMappers,
  telegram: telegramComponentMappers,
  octopus: octopusComponentMappers,
  teams: teamsComponentMappers,
  openai: openaiComponentMappers,
  circleci: circleCIComponentMappers,
  claude: claudeComponentMappers,
  logfire: logfireComponentMappers,
  perplexity: perplexityComponentMappers,
  gcp: gcpComponentMappers,
  prometheus: prometheusComponentMappers,
  cursor: cursorComponentMappers,
  hetzner: hetznerComponentMappers,
  jfrogArtifactory: jfrogArtifactoryComponentMappers,
  statuspage: statuspageComponentMappers,
  dockerhub: dockerhubComponentMappers,
  honeycomb: honeycombComponentMappers,
  harness: harnessComponentMappers,
  servicenow: servicenowComponentMappers,
  elastic: elasticComponentMappers,
  oci: ociComponentMappers,
};

const appTriggerRenderers: Record<string, Record<string, TriggerRenderer>> = {
  cloudflare: cloudflareTriggerRenderers,
  digitalocean: digitaloceanTriggerRenderers,
  semaphore: semaphoreTriggerRenderers,
  github: githubTriggerRenderers,
  gitlab: gitlabTriggerRenderers,
  pagerduty: pagerdutyTriggerRenderers,
  dash0: dash0TriggerRenderers,
  daytona: daytonaTriggerRenderers,
  datadog: datadogTriggerRenderers,
  slack: slackTriggerRenderers,
  smtp: smtpTriggerRenderers,
  sendgrid: sendgridTriggerRenderers,
  sentry: sentryTriggerRenderers,
  render: renderTriggerRenderers,
  rootly: rootlyTriggerRenderers,
  incident: incidentTriggerRenderers,
  newrelic: newrelicTriggerRenderers,
  firehydrant: firehydrantTriggerRenderers,
  launchdarkly: launchdarklyTriggerRenderers,
  aws: awsTriggerRenderers,
  azure: azureTriggerRenderers,
  discord: discordTriggerRenderers,
  telegram: telegramTriggerRenderers,
  octopus: octopusTriggerRenderers,
  teams: teamsTriggerRenderers,
  openai: openaiTriggerRenderers,
  circleci: circleCITriggerRenderers,
  claude: claudeTriggerRenderers,
  logfire: logfireTriggerRenderers,
  perplexity: perplexityTriggerRenderers,
  gcp: gcpTriggerRenderers,
  grafana: grafanaTriggerRenderers,
  bitbucket: bitbucketTriggerRenderers,
  prometheus: prometheusTriggerRenderers,
  cursor: cursorTriggerRenderers,
  jfrogArtifactory: jfrogArtifactoryTriggerRenderers,
  statuspage: statuspageTriggerRenderers,
  dockerhub: dockerhubTriggerRenderers,
  honeycomb: honeycombTriggerRenderers,
  harness: harnessTriggerRenderers,
  servicenow: servicenowTriggerRenderers,
  elastic: elasticTriggerRenderers,
  oci: ociTriggerRenderers,
};

const appEventStateRegistries: Record<string, Record<string, EventStateRegistry>> = {
  cloudflare: cloudflareEventStateRegistry,
  digitalocean: digitaloceanEventStateRegistry,
  semaphore: semaphoreEventStateRegistry,
  github: githubEventStateRegistry,
  pagerduty: pagerdutyEventStateRegistry,
  dash0: dash0EventStateRegistry,
  daytona: daytonaEventStateRegistry,
  datadog: datadogEventStateRegistry,
  slack: slackEventStateRegistry,
  smtp: smtpEventStateRegistry,
  sendgrid: sendgridEventStateRegistry,
  sentry: sentryEventStateRegistry,
  render: renderEventStateRegistry,
  discord: discordEventStateRegistry,
  telegram: telegramEventStateRegistry,
  teams: teamsEventStateRegistry,
  rootly: rootlyEventStateRegistry,
  incident: incidentEventStateRegistry,
  newrelic: newrelicEventStateRegistry,
  octopus: octopusEventStateRegistry,
  firehydrant: firehydrantEventStateRegistry,
  launchdarkly: launchdarklyEventStateRegistry,
  openai: openaiEventStateRegistry,
  circleci: circleCIEventStateRegistry,
  claude: claudeEventStateRegistry,
  logfire: logfireEventStateRegistry,
  perplexity: perplexityEventStateRegistry,
  gcp: gcpEventStateRegistry,
  statuspage: statuspageEventStateRegistry,
  aws: awsEventStateRegistry,
  grafana: grafanaEventStateRegistry,
  prometheus: prometheusEventStateRegistry,
  cursor: cursorEventStateRegistry,
  azure: azureEventStateRegistry,
  gitlab: gitlabEventStateRegistry,
  jfrogArtifactory: jfrogArtifactoryEventStateRegistry,
  dockerhub: dockerhubEventStateRegistry,
  honeycomb: honeycombEventStateRegistry,
  harness: harnessEventStateRegistry,
  servicenow: servicenowEventStateRegistry,
  elastic: elasticEventStateRegistry,
  oci: ociEventStateRegistry,
};

const eventStateRegistries: Record<string, EventStateRegistry> = {
  approval: APPROVAL_STATE_REGISTRY,
  http: HTTP_STATE_REGISTRY,
  graphql: GRAPHQL_STATE_REGISTRY,
  ssh: SSH_STATE_REGISTRY,
  runner: RUNNER_STATE_REGISTRY,
  filter: FILTER_STATE_REGISTRY,
  if: IF_STATE_REGISTRY,
  timeGate: TIME_GATE_STATE_REGISTRY,
  wait: WAIT_STATE_REGISTRY,
  merge: MERGE_STATE_REGISTRY,
  sendEmail: SEND_EMAIL_STATE_REGISTRY,
};

const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  schedule: scheduleCustomFieldRenderer,
  wait: waitCustomFieldRenderer,
  webhook: webhookCustomFieldRenderer,
};

const appCustomFieldRenderers: Record<string, Record<string, CustomFieldRenderer>> = {
  grafana: grafanaCustomFieldRenderers,
  newrelic: newrelicCustomFieldRenderers,
  prometheus: prometheusCustomFieldRenderers,
  dockerhub: dockerhubCustomFieldRenderers,
  incident: incidentCustomFieldRenderers,
  gcp: gcpCustomFieldRenderers,
  servicenow: servicenowCustomFieldRenderers,
};

/**
 * Get the appropriate renderer for a trigger type.
 * Falls back to the default renderer if no specific renderer is registered.
 * The returned renderer is wrapped in a safe wrapper that catches exceptions
 * to prevent a single trigger mapper failure from breaking the entire canvas.
 */
export function getTriggerRenderer(name: string): TriggerRenderer {
  if (!name) {
    return createSafeTriggerRenderer(defaultTriggerRenderer, name || "default");
  }

  const parts = name?.split(".");
  if (parts?.length === 1) {
    return createSafeTriggerRenderer(withCustomName(triggerRenderers[name] || defaultTriggerRenderer), name);
  }

  const appName = parts[0];
  const appTriggers = appTriggerRenderers[appName];
  if (!appTriggers) {
    return createSafeTriggerRenderer(withCustomName(defaultTriggerRenderer), name);
  }

  const triggerName = parts.slice(1).join(".");
  return createSafeTriggerRenderer(withCustomName(appTriggers[triggerName] || defaultTriggerRenderer), name);
}

/**
 * Get the appropriate mapper for a component.
 * Falls back to the noop mapper if no specific mapper is registered.
 * The returned mapper is wrapped in a safe wrapper that catches exceptions
 * to prevent a single component mapper failure from breaking the entire canvas.
 */
export function getComponentBaseMapper(name: string): ComponentBaseMapper {
  return createSafeComponentMapper(findRegisteredComponentMapper(name) || noopMapper, name || "noop");
}

/**
 * Get the appropriate state registry for a component type.
 * Falls back to the default state registry if no specific registry is registered.
 */
export function getEventStateRegistry(name: string): EventStateRegistry {
  const parts = name.split(".");
  if (parts.length === 1) {
    return eventStateRegistries[name] || DEFAULT_STATE_REGISTRY;
  }

  const appName = parts[0];
  const appRegistry = appEventStateRegistries[appName];
  if (!appRegistry) {
    return DEFAULT_STATE_REGISTRY;
  }

  const componentName = parts.slice(1).join(".");
  return appRegistry[componentName] || DEFAULT_STATE_REGISTRY;
}

/**
 * Get the state map for a component type.
 * Falls back to the default state map if no specific registry is registered.
 */
export function getStateMap(componentName: string) {
  return getEventStateRegistry(componentName).stateMap;
}

/**
 * Get the state function for a component type.
 * Falls back to the default state function if no specific registry is registered.
 */
export function getState(componentName: string) {
  return getEventStateRegistry(componentName).getState;
}

/**
 * Get the appropriate custom field renderer for a component/trigger type.
 * Returns undefined if no specific renderer is registered.
 */
export function getCustomFieldRenderer(componentName: string): CustomFieldRenderer | undefined {
  const parts = componentName?.split(".");
  if (parts?.length === 1) {
    const renderer = customFieldRenderers[componentName];
    return renderer ? createSafeCustomFieldRenderer(renderer, componentName) : undefined;
  }

  const appName = parts[0];
  const appRenderers = appCustomFieldRenderers[appName];
  if (!appRenderers) {
    return undefined;
  }

  const name = parts.slice(1).join(".");
  const renderer = appRenderers[name];
  return renderer ? createSafeCustomFieldRenderer(renderer, componentName) : undefined;
}

/**
 * Get the execution details for a component execution.
 * Returns undefined if no specific execution details function is registered.
 * Catches exceptions from mappers to prevent canvas-wide failures.
 */
export function getExecutionDetails(
  componentName: string,
  execution: CanvasesCanvasNodeExecution,
  node: ComponentsNode,
  nodes?: ComponentsNode[],
): Record<string, unknown> | undefined {
  const mapper = findRegisteredComponentMapper(componentName);
  if (!mapper) {
    return undefined;
  }

  return createSafeComponentMapper(mapper, componentName).getExecutionDetails({
    execution: buildExecutionInfo(execution),
    node: buildNodeInfo(node),
    nodes: nodes?.map((n) => buildNodeInfo(n)) || [],
  });
}

function findRegisteredComponentMapper(name: string): ComponentBaseMapper | undefined {
  const parts = name?.split(".");
  if (!parts?.length) {
    return undefined;
  }

  if (parts.length === 1) {
    return componentBaseMappers[name];
  }

  const appMapper = appMappers[parts[0]];
  if (!appMapper) {
    return undefined;
  }

  return appMapper[parts.slice(1).join(".")];
}

function withCustomName(renderer: TriggerRenderer): TriggerRenderer {
  return {
    ...renderer,
    getTriggerProps: (context: TriggerRendererContext) => {
      const props = renderer.getTriggerProps(context);
      const customName = context.lastEvent?.customName?.trim();
      if (customName && props.lastEventData) {
        return {
          ...props,
          lastEventData: {
            ...props.lastEventData,
            title: customName,
          },
        };
      }

      return props;
    },
    getTitleAndSubtitle: (context: TriggerEventContext) => {
      const { title, subtitle } = renderer.getTitleAndSubtitle(context);
      const customName = context.event?.customName?.trim();
      if (customName) {
        return { title: customName, subtitle };
      }

      return { title, subtitle };
    },
  };
}
