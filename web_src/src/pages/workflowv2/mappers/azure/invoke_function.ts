import type { ComponentBaseContext, ComponentBaseMapper, SubtitleContext } from "../types";
import { noopMapper } from "../noop";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

type InvokeFunctionConfiguration = {
  functionApp?: unknown;
  functionName?: unknown;
  httpMethod?: unknown;
};

function getStringValue(value: unknown): string | undefined {
  if (typeof value === "string" && value.trim().length > 0) {
    return value.trim();
  }
  if (value && typeof value === "object") {
    const obj = value as Record<string, unknown>;
    for (const field of ["label", "name", "value"]) {
      const candidate = obj[field];
      if (typeof candidate === "string" && candidate.trim().length > 0) {
        return candidate.trim();
      }
    }
  }
  return undefined;
}

function metadataList(context: ComponentBaseContext): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = (context.node.configuration as InvokeFunctionConfiguration | undefined) ?? {};

  const functionApp = getStringValue(config.functionApp);
  const functionName = getStringValue(config.functionName);
  const httpMethod = getStringValue(config.httpMethod);

  if (functionApp) {
    metadata.push({ icon: "server", label: functionApp });
  }
  if (functionName) {
    metadata.push({ icon: "zap", label: functionName });
  }
  if (httpMethod) {
    metadata.push({ icon: "arrow-right", label: httpMethod });
  }

  return metadata;
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  if (!context.execution.createdAt) return "";
  return renderTimeAgo(new Date(context.execution.createdAt));
}

export const invokeFunctionMapper: ComponentBaseMapper = {
  ...noopMapper,
  props: (context) => ({
    ...noopMapper.props(context),
    metadata: metadataList(context),
  }),
  subtitle,
};
