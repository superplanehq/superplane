import { ComponentBaseMapper, EventStateRegistry } from "../types";
import { DEFAULT_STATE_REGISTRY } from "../stateRegistry";
import { launchAgentMapper } from "./launch_agent";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  launchAgent: launchAgentMapper,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  launchAgent: DEFAULT_STATE_REGISTRY,
};
