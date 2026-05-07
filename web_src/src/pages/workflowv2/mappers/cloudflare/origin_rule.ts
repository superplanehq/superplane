import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface OriginRuleMatchRule {
  field?: string;
  operator?: string;
  value?: string;
  conjunction?: string;
}

interface OriginRuleConfiguration {
  zone?: string;
  rule?: string;
  matchMode?: "custom" | "all" | string;
  matchRules?: OriginRuleMatchRule[];
  expression?: string;
  originHost?: string | null;
  originPort?: number | null;
  hostHeader?: string | null;
  sni?: string | null;
  enabled?: boolean;
}

interface OriginRuleNodeMetadata extends OriginRuleConfiguration {
  zoneName?: string;
  rewrites?: string[];
}

interface OriginRuleOutput {
  id?: string;
  zoneId?: string;
  ruleId?: string;
  rule?: {
    id?: string;
    expression?: string;
    enabled?: boolean;
    action_parameters?: {
      host_header?: string;
      origin?: {
        host?: string;
        port?: number;
      };
      sni?: {
        value?: string;
      };
    };
  };
}

export const originRuleMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      metadata: originRuleMetadataList(context.node),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as OriginRuleConfiguration | undefined;
    const nodeMetadata = context.node.metadata as OriginRuleNodeMetadata | undefined;
    const output = getOriginRuleOutput(context);
    const rule = output?.rule;
    const actionParameters = rule?.action_parameters;
    const details: Record<string, string> = {};

    details["Executed At"] = executionTimestamp(context);
    addDetail(details, "Rule ID", stringOrDash(output?.ruleId || output?.id || rule?.id || configuration?.rule));
    addDetail(details, "Zone", stringOrDash(nodeMetadata?.zoneName));
    addDetail(details, "Match", originRuleMatchLabel(configuration, nodeMetadata, rule?.expression));
    addDetail(
      details,
      "DNS Record",
      stringOrDash(actionParameters?.origin?.host || configuration?.originHost || nodeMetadata?.originHost),
    );
    addDetail(
      details,
      "Host Header",
      stringOrDash(actionParameters?.host_header || configuration?.hostHeader || nodeMetadata?.hostHeader),
    );
    addDetail(details, "SNI", stringOrDash(actionParameters?.sni?.value || configuration?.sni || nodeMetadata?.sni));
    addDetail(
      details,
      "Destination Port",
      stringOrDash(actionParameters?.origin?.port ?? configuration?.originPort ?? nodeMetadata?.originPort),
    );
    addDetail(details, "Enabled", booleanLabel(rule?.enabled ?? configuration?.enabled ?? nodeMetadata?.enabled));

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return baseMapper.subtitle(context);
  },
};

function originRuleMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as OriginRuleConfiguration | undefined;
  const nodeMetadata = node.metadata as OriginRuleNodeMetadata | undefined;

  const zoneName = nodeMetadata?.zoneName;
  const rule = configuration?.rule || nodeMetadata?.rule;
  if (zoneName) {
    metadata.push({ icon: "globe", label: zoneName });
  } else if (rule) {
    metadata.push({ icon: "route", label: truncate(rule, 72) });
  }

  const match = originRuleMatchLabel(configuration, nodeMetadata);
  if (match !== "-") {
    metadata.push({ icon: "list-filter", label: match });
  }

  const originHost = configuration?.originHost || nodeMetadata?.originHost;
  if (originHost) {
    metadata.push({ icon: "server", label: `DNS: ${originHost}` });
  }

  const rewrites = originRuleRewriteLabels(configuration, nodeMetadata);
  if (rewrites.length > 0) {
    metadata.push({ icon: "shuffle", label: rewrites.join(", ") });
  }

  const enabled = configuration?.enabled ?? nodeMetadata?.enabled;
  if (enabled !== undefined) {
    metadata.push({ icon: "power", label: enabled ? "Enabled" : "Disabled" });
  }

  return metadata.slice(0, 3);
}

function getOriginRuleOutput(context: ExecutionDetailsContext): OriginRuleOutput | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const data = outputs?.default?.[0]?.data;
  if (!data || typeof data !== "object") {
    return undefined;
  }

  return data as OriginRuleOutput;
}

function originRuleMatchLabel(
  configuration?: OriginRuleConfiguration,
  nodeMetadata?: OriginRuleNodeMetadata,
  expression?: string,
): string {
  if (configuration?.matchMode === "all" || nodeMetadata?.matchMode === "all" || expression === "true") {
    return "All incoming requests";
  }

  const rules = configuration?.matchRules;
  if (Array.isArray(rules) && rules.length > 0) {
    const preview = rules.slice(0, 2).map(formatMatchRule).filter(Boolean);
    if (preview.length > 0) {
      const suffix = rules.length > 2 ? ` +${rules.length - 2}` : "";
      return `${preview.join(" / ")}${suffix}`;
    }
  }

  const resolvedExpression = expression || nodeMetadata?.expression || configuration?.expression;
  if (resolvedExpression) {
    return truncate(resolvedExpression, 72);
  }

  return "-";
}

function formatMatchRule(rule: OriginRuleMatchRule): string {
  const field = originRuleFieldLabel(rule.field);
  const operator = originRuleOperatorLabel(rule.operator);
  const value = rule.value || "";
  if (!field || !operator || !value) {
    return "";
  }

  return `${field} ${operator} ${value}`;
}

function originRuleFieldLabel(field?: string): string {
  switch (field) {
    case "fullUri":
      return "URL Full";
    case "uriPath":
      return "URI Path";
    case "host":
      return "Hostname";
    case "query":
      return "URI Query";
    case "method":
      return "HTTP Method";
    case "scheme":
      return "Scheme";
    default:
      return field || "";
  }
}

function originRuleOperatorLabel(operator?: string): string {
  switch (operator) {
    case "wildcard":
      return "wildcard";
    case "equals":
      return "=";
    case "notEquals":
      return "!=";
    case "contains":
      return "contains";
    case "startsWith":
      return "starts with";
    case "endsWith":
      return "ends with";
    case "matches":
      return "matches";
    default:
      return operator || "";
  }
}

function originRuleRewriteLabels(
  configuration?: OriginRuleConfiguration,
  nodeMetadata?: OriginRuleNodeMetadata,
): string[] {
  const fromMetadata = nodeMetadata?.rewrites?.filter(Boolean);
  if (fromMetadata && fromMetadata.length > 0) {
    return fromMetadata;
  }

  const rewrites: string[] = [];
  if (configuration?.originHost) rewrites.push("DNS Record");
  if (configuration?.hostHeader) rewrites.push("Host Header");
  if (configuration?.sni) rewrites.push("SNI");
  if (configuration?.originPort) rewrites.push("Destination Port");
  return rewrites;
}

function addDetail(details: Record<string, string>, key: string, value: string): void {
  if (Object.keys(details).length >= 6 || value === "-") {
    return;
  }

  details[key] = value;
}

function executionTimestamp(context: ExecutionDetailsContext): string {
  const timestamp = context.execution.createdAt || context.execution.updatedAt;
  return timestamp ? new Date(timestamp).toLocaleString() : "-";
}

function booleanLabel(value?: boolean): string {
  if (value === undefined) {
    return "-";
  }

  return value ? "Yes" : "No";
}

function stringOrDash(value: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}

function truncate(value: string, maxLength: number): string {
  return value.length > maxLength ? `${value.slice(0, maxLength)}...` : value;
}
