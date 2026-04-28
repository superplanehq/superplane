import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { invokeHandlerMapper } from "./invoke_handler";
import { sendHandlerMapper } from "./send_handler";
import { registerDeploymentMapper } from "./register_deployment";
import { removeDeploymentMapper } from "./remove_deployment";
import { getServiceMapper } from "./get_service";
import { listServicesMapper } from "./list_services";
import { cancelInvocationMapper, killInvocationMapper, purgeInvocationMapper } from "./invocation_action";
import { healthCheckMapper } from "./health_check";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  invokeHandler: invokeHandlerMapper,
  sendHandler: sendHandlerMapper,
  sendDelayedHandler: sendHandlerMapper, // reuses send mapper (same output shape)
  registerDeployment: registerDeploymentMapper,
  removeDeployment: removeDeploymentMapper,
  getService: getServiceMapper,
  listServices: listServicesMapper,
  cancelInvocation: cancelInvocationMapper,
  killInvocation: killInvocationMapper,
  purgeInvocation: purgeInvocationMapper,
  healthCheck: healthCheckMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  invokeHandler: buildActionStateRegistry("Handler invoked"),
  sendHandler: buildActionStateRegistry("Handler sent"),
  sendDelayedHandler: buildActionStateRegistry("Delayed handler sent"),
  registerDeployment: buildActionStateRegistry("Deployment registered"),
  removeDeployment: buildActionStateRegistry("Deployment removed"),
  getService: buildActionStateRegistry("Service retrieved"),
  listServices: buildActionStateRegistry("Services listed"),
  cancelInvocation: buildActionStateRegistry("Invocation cancelled"),
  killInvocation: buildActionStateRegistry("Invocation killed"),
  purgeInvocation: buildActionStateRegistry("Invocation purged"),
  healthCheck: buildActionStateRegistry("Health checked"),
};
