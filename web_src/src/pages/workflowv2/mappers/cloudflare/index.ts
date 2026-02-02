import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDnsRecord: baseMapper,
  updateDNSRecord: baseMapper,
  deleteDnsRecord: baseMapper,
  updateRedirectRule: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDnsRecord: buildActionStateRegistry("completed"),
  updateDNSRecord: buildActionStateRegistry("completed"),
  deleteDnsRecord: buildActionStateRegistry("completed"),
  updateRedirectRule: buildActionStateRegistry("completed"),
};
