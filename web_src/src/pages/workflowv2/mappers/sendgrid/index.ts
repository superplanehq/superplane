import { ComponentBaseMapper, EventStateRegistry, ExecutionInfo, StateFunction, TriggerRenderer } from "../types";
import { sendEmailMapper } from "./send_email";
import { DEFAULT_EVENT_STATE_MAP, EventState, EventStateMap } from "@/ui/componentBase";
import { defaultStateFunction } from "../stateRegistry";
import { onEmailEventTriggerRenderer } from "./on_email_event";
import { createOrUpdateContactMapper } from "./create_or_update_contact";

export const SEND_EMAIL_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const sendEmailStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: { data?: unknown }[] } | undefined;
  if (outputs?.failed?.length) {
    return "failed";
  }

  return defaultStateFunction(execution);
};

export const SEND_EMAIL_STATE_REGISTRY: EventStateRegistry = {
  stateMap: SEND_EMAIL_STATE_MAP,
  getState: sendEmailStateFunction,
};

export const CREATE_OR_UPDATE_CONTACT_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const createOrUpdateContactStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: { data?: unknown }[] } | undefined;
  if (outputs?.failed?.length) {
    return "failed";
  }

  return defaultStateFunction(execution);
};

export const CREATE_OR_UPDATE_CONTACT_STATE_REGISTRY: EventStateRegistry = {
  stateMap: CREATE_OR_UPDATE_CONTACT_STATE_MAP,
  getState: createOrUpdateContactStateFunction,
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendEmail: sendEmailMapper,
  createOrUpdateContact: createOrUpdateContactMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onEmailEvent: onEmailEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendEmail: SEND_EMAIL_STATE_REGISTRY,
  createOrUpdateContact: CREATE_OR_UPDATE_CONTACT_STATE_REGISTRY,
};
