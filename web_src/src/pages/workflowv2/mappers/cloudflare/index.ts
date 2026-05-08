import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { originRuleMapper } from "./origin_rule";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDnsRecord: baseMapper,
  createOriginRule: originRuleMapper,
  updateDNSRecord: baseMapper,
  deleteDnsRecord: baseMapper,
  updateRedirectRule: baseMapper,
  updateOriginRule: originRuleMapper,
  deleteOriginRule: originRuleMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDnsRecord: buildActionStateRegistry("created"),
  createOriginRule: buildActionStateRegistry("created"),
  updateDNSRecord: buildActionStateRegistry("updated"),
  deleteDnsRecord: buildActionStateRegistry("deleted"),
  updateRedirectRule: buildActionStateRegistry("updated"),
  updateOriginRule: buildActionStateRegistry("updated"),
  deleteOriginRule: buildActionStateRegistry("deleted"),
};
