import {
  ComponentBaseMapper,
  TriggerRenderer,
  ComponentAdditionalDataBuilder,
  EventStateRegistry,
  CustomFieldRenderer,
  TriggerRendererContext,
  TriggerEventContext,
} from "./types";
import { ComponentsNode, CanvasesCanvasNodeExecution } from "@/api-client";
import { defaultTriggerRenderer } from "./default";
import { scheduleTriggerRenderer, scheduleCustomFieldRenderer } from "./schedule";
import { webhookTriggerRenderer, webhookCustomFieldRenderer } from "./webhook";
import { noopMapper } from "./noop";
import { ifMapper, IF_STATE_REGISTRY } from "./if";
import { httpMapper, HTTP_STATE_REGISTRY } from "./http";
import {
  componentMappers as semaphoreComponentMappers,
  triggerRenderers as semaphoreTriggerRenderers,
  eventStateRegistry as semaphoreEventStateRegistry,
} from "./semaphore/index";
import {
  componentMappers as githubComponentMappers,
  triggerRenderers as githubTriggerRenderers,
  eventStateRegistry as githubEventStateRegistry,
  customFieldRenderers as githubCustomFieldRenderers,
} from "./github/index";
import {
  componentMappers as gitlabComponentMappers,
  triggerRenderers as gitlabTriggerRenderers,
  eventStateRegistry as gitlabEventStateRegistry,
} from "./gitlab/index";
import {
  componentMappers as pagerdutyComponentMappers,
  triggerRenderers as pagerdutyTriggerRenderers,
  eventStateRegistry as pagerdutyEventStateRegistry,
} from "./pagerduty/index";
import {
  componentMappers as dash0ComponentMappers,
  triggerRenderers as dash0TriggerRenderers,
  eventStateRegistry as dash0EventStateRegistry,
} from "./dash0/index";
import {
  componentMappers as daytonaComponentMappers,
  triggerRenderers as daytonaTriggerRenderers,
  eventStateRegistry as daytonaEventStateRegistry,
} from "./daytona/index";
import {
  componentMappers as cloudflareComponentMappers,
  triggerRenderers as cloudflareTriggerRenderers,
  eventStateRegistry as cloudflareEventStateRegistry,
} from "./cloudflare/index";
import {
  componentMappers as datadogComponentMappers,
  triggerRenderers as datadogTriggerRenderers,
  eventStateRegistry as datadogEventStateRegistry,
} from "./datadog/index";
import {
  componentMappers as slackComponentMappers,
  triggerRenderers as slackTriggerRenderers,
  eventStateRegistry as slackEventStateRegistry,
} from "./slack";
import {
  componentMappers as smtpComponentMappers,
  triggerRenderers as smtpTriggerRenderers,
  eventStateRegistry as smtpEventStateRegistry,
} from "./smtp";
import {
  componentMappers as sendgridComponentMappers,
  triggerRenderers as sendgridTriggerRenderers,
  eventStateRegistry as sendgridEventStateRegistry,
} from "./sendgrid";
import {
  componentMappers as renderComponentMappers,
  triggerRenderers as renderTriggerRenderers,
  eventStateRegistry as renderEventStateRegistry,
} from "./render";
import {
  componentMappers as rootlyComponentMappers,
  triggerRenderers as rootlyTriggerRenderers,
  eventStateRegistry as rootlyEventStateRegistry,
} from "./rootly/index";
import {
  componentMappers as awsComponentMappers,
  triggerRenderers as awsTriggerRenderers,
  eventStateRegistry as awsEventStateRegistry,
} from "./aws";
import { triggerRenderers as bitbucketTriggerRenderers } from "./bitbucket/index";
import { componentMappers as hetznerComponentMappers } from "./hetzner/index";
import { timeGateMapper, TIME_GATE_STATE_REGISTRY } from "./timegate";
import {
  componentMappers as digitaloceanComponentMappers,
  triggerRenderers as digitaloceanTriggerRenderers,
  eventStateRegistry as digitaloceanEventStateRegistry,
} from "./digitalocean/index";
import {
  componentMappers as discordComponentMappers,
  triggerRenderers as discordTriggerRenderers,
  eventStateRegistry as discordEventStateRegistry,
} from "./discord";
import {
  componentMappers as openaiComponentMappers,
  triggerRenderers as openaiTriggerRenderers,
  eventStateRegistry as openaiEventStateRegistry,
} from "./openai/index";
import {
  componentMappers as grafanaComponentMappers,
  customFieldRenderers as grafanaCustomFieldRenderers,
  triggerRenderers as grafanaTriggerRenderers,
  eventStateRegistry as grafanaEventStateRegistry,
} from "./grafana/index";
import {
  componentMappers as circleCIComponentMappers,
  triggerRenderers as circleCITriggerRenderers,
  eventStateRegistry as circleCIEventStateRegistry,
} from "./circleci/index";
import {
  componentMappers as claudeComponentMappers,
  triggerRenderers as claudeTriggerRenderers,
  eventStateRegistry as claudeEventStateRegistry,
} from "./claude/index";
import {
  componentMappers as prometheusComponentMappers,
  customFieldRenderers as prometheusCustomFieldRenderers,
  triggerRenderers as prometheusTriggerRenderers,
  eventStateRegistry as prometheusEventStateRegistry,
} from "./prometheus/index";
import {
  componentMappers as cursorComponentMappers,
  triggerRenderers as cursorTriggerRenderers,
  eventStateRegistry as cursorEventStateRegistry,
} from "./cursor/index";
import {
  componentMappers as statuspageComponentMappers,
  triggerRenderers as statuspageTriggerRenderers,
  eventStateRegistry as statuspageEventStateRegistry,
} from "./statuspage";
import {
  componentMappers as dockerhubComponentMappers,
  customFieldRenderers as dockerhubCustomFieldRenderers,
  triggerRenderers as dockerhubTriggerRenderers,
  eventStateRegistry as dockerhubEventStateRegistry,
} from "./dockerhub";
import { filterMapper, FILTER_STATE_REGISTRY } from "./filter";
import { sshMapper, SSH_STATE_REGISTRY } from "./ssh";
import { waitCustomFieldRenderer, waitMapper, WAIT_STATE_REGISTRY } from "./wait";
import { approvalMapper, approvalDataBuilder, APPROVAL_STATE_REGISTRY } from "./approval";
import { mergeMapper, MERGE_STATE_REGISTRY } from "./merge";
import { DEFAULT_STATE_REGISTRY } from "./stateRegistry";
import { startTriggerRenderer } from "./start";
import { buildExecutionInfo, buildNodeInfo } from "../utils";

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
  if: ifMapper,
  http: httpMapper,
  ssh: sshMapper,
  timeGate: timeGateMapper,
  filter: filterMapper,
  wait: waitMapper,
  approval: approvalMapper,
  merge: mergeMapper,
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
  render: renderComponentMappers,
  rootly: rootlyComponentMappers,
  aws: awsComponentMappers,
  discord: discordComponentMappers,
  openai: openaiComponentMappers,
  circleci: circleCIComponentMappers,
  claude: claudeComponentMappers,
  prometheus: prometheusComponentMappers,
  cursor: cursorComponentMappers,
  hetzner: hetznerComponentMappers,
  statuspage: statuspageComponentMappers,
  dockerhub: dockerhubComponentMappers,
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
  render: renderTriggerRenderers,
  rootly: rootlyTriggerRenderers,
  aws: awsTriggerRenderers,
  discord: discordTriggerRenderers,
  openai: openaiTriggerRenderers,
  circleci: circleCITriggerRenderers,
  claude: claudeTriggerRenderers,
  grafana: grafanaTriggerRenderers,
  bitbucket: bitbucketTriggerRenderers,
  prometheus: prometheusTriggerRenderers,
  cursor: cursorTriggerRenderers,
  statuspage: statuspageTriggerRenderers,
  dockerhub: dockerhubTriggerRenderers,
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
  render: renderEventStateRegistry,
  discord: discordEventStateRegistry,
  rootly: rootlyEventStateRegistry,
  openai: openaiEventStateRegistry,
  circleci: circleCIEventStateRegistry,
  claude: claudeEventStateRegistry,
  statuspage: statuspageEventStateRegistry,
  aws: awsEventStateRegistry,
  grafana: grafanaEventStateRegistry,
  prometheus: prometheusEventStateRegistry,
  cursor: cursorEventStateRegistry,
  gitlab: gitlabEventStateRegistry,
  dockerhub: dockerhubEventStateRegistry,
};

const componentAdditionalDataBuilders: Record<string, ComponentAdditionalDataBuilder> = {
  approval: approvalDataBuilder,
};

const eventStateRegistries: Record<string, EventStateRegistry> = {
  approval: APPROVAL_STATE_REGISTRY,
  http: HTTP_STATE_REGISTRY,
  ssh: SSH_STATE_REGISTRY,
  filter: FILTER_STATE_REGISTRY,
  if: IF_STATE_REGISTRY,
  timeGate: TIME_GATE_STATE_REGISTRY,
  wait: WAIT_STATE_REGISTRY,
  merge: MERGE_STATE_REGISTRY,
};

const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  schedule: scheduleCustomFieldRenderer,
  wait: waitCustomFieldRenderer,
  webhook: webhookCustomFieldRenderer,
};

const appCustomFieldRenderers: Record<string, Record<string, CustomFieldRenderer>> = {
  github: githubCustomFieldRenderers,
  grafana: grafanaCustomFieldRenderers,
  prometheus: prometheusCustomFieldRenderers,
  dockerhub: dockerhubCustomFieldRenderers,
};

/**
 * Get the appropriate renderer for a trigger type.
 * Falls back to the default renderer if no specific renderer is registered.
 */
export function getTriggerRenderer(name: string): TriggerRenderer {
  if (!name) {
    return defaultTriggerRenderer;
  }

  const parts = name?.split(".");
  if (parts?.length == 1) {
    return withCustomName(triggerRenderers[name] || defaultTriggerRenderer);
  }

  const appName = parts[0];
  const appTriggers = appTriggerRenderers[appName];
  if (!appTriggers) {
    return withCustomName(defaultTriggerRenderer);
  }

  const triggerName = parts.slice(1).join(".");
  return withCustomName(appTriggers[triggerName] || defaultTriggerRenderer);
}

/**
 * Get the appropriate mapper for a component.
 * Falls back to the noop mapper if no specific mapper is registered.
 */
export function getComponentBaseMapper(name: string): ComponentBaseMapper {
  const parts = name?.split(".");
  if (parts?.length == 1) {
    return componentBaseMappers[name] || noopMapper;
  }

  const appName = parts[0];
  const appMapper = appMappers[appName];
  if (!appMapper) {
    return noopMapper;
  }

  const componentName = parts.slice(1).join(".");
  return appMapper[componentName] || noopMapper;
}

/**
 * Get the appropriate additional data builder for a component type.
 * Returns undefined if no specific builder is registered.
 */
export function getComponentAdditionalDataBuilder(componentName: string): ComponentAdditionalDataBuilder | undefined {
  return componentAdditionalDataBuilders[componentName];
}

/**
 * Get the appropriate state registry for a component type.
 * Falls back to the default state registry if no specific registry is registered.
 */
export function getEventStateRegistry(name: string): EventStateRegistry {
  const parts = name.split(".");
  if (parts.length == 1) {
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
    return customFieldRenderers[componentName];
  }

  const appName = parts[0];
  const appRenderers = appCustomFieldRenderers[appName];
  if (!appRenderers) {
    return undefined;
  }

  const name = parts.slice(1).join(".");
  return appRenderers[name];
}

/**
 * Get the execution details for a component execution.
 * Returns undefined if no specific execution details function is registered.
 */
export function getExecutionDetails(
  componentName: string,
  execution: CanvasesCanvasNodeExecution,
  node: ComponentsNode,
  nodes?: ComponentsNode[],
): Record<string, any> | undefined {
  const parts = componentName?.split(".");
  let mapper: ComponentBaseMapper | undefined;

  if (parts?.length === 1) {
    mapper = componentBaseMappers[componentName];
  } else {
    const appName = parts[0];
    const appMapper = appMappers[appName];
    if (appMapper) {
      const componentNamePart = parts.slice(1).join(".");
      mapper = appMapper[componentNamePart];
    }
  }

  return mapper?.getExecutionDetails?.({
    execution: buildExecutionInfo(execution),
    node: buildNodeInfo(node),
    nodes: nodes?.map((n) => buildNodeInfo(n)) || [],
  });
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
