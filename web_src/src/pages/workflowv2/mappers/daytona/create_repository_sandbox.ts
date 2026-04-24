import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper } from "./base";
import type { ComponentBaseSpec } from "@/ui/componentBase";
import { formatDuration } from "@/lib/duration";

interface CreateRepositorySandboxConfiguration {
  snapshot?: string;
  target?: string;
  repository?: string;
  bootstrap?: {
    from?: string;
    script?: string;
    path?: string;
    timeout?: number;
  };
}

interface CreateRepositorySandboxMetadata {
  stage?: string;
  sandboxId?: string;
  sandboxStartedAt?: string;
  sessionId?: string;
  timeout?: number;
  repository?: string;
  directory?: string;
  clone?: {
    cmdId?: string;
  };
  bootstrap?: {
    cmdId?: string;
    startedAt?: string;
    finishedAt?: string;
    exitCode?: number;
    result?: string;
    log?: string;
  };
}

export const createRepositorySandboxMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    return {
      ...props,
      metadata: createRepositorySandboxMetadataList(context.node),
      specs: createRepositorySandboxSpecs(context.node),
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const metadata = context.execution.metadata as CreateRepositorySandboxMetadata | undefined;
    const details: Record<string, string> = {};

    if (metadata?.stage) {
      details["Step"] = metadata.stage;
    }

    if (metadata?.sandboxId) {
      details["Sandbox ID"] = metadata.sandboxId;
    }
    if (metadata?.repository) {
      details["Repository"] = metadata.repository;
    }
    if (metadata?.directory) {
      details["Directory"] = metadata.directory;
    }

    const elapsedLabel = buildElapsedLabel(context, metadata);
    if (elapsedLabel) {
      details["Elapsed"] = elapsedLabel;
    }

    if (metadata?.bootstrap?.log) {
      details["Bootstrap log"] = metadata.bootstrap.log;
    }

    return details;
  },
};

function buildElapsedLabel(
  context: ExecutionDetailsContext,
  metadata: CreateRepositorySandboxMetadata | undefined,
): string | undefined {
  if (!metadata?.sandboxStartedAt) {
    return undefined;
  }

  const timeoutMs = metadata.timeout ? metadata.timeout * 1000 : undefined;
  const startedAtMs = Date.parse(metadata.sandboxStartedAt);
  if (Number.isNaN(startedAtMs)) {
    return undefined;
  }

  // For finished runs, show elapsed frozen at execution end when available;
  // for in-flight runs, elapsed ticks forward naturally on each sidebar refresh
  // (the sidebar already polls every 1.5s).
  const endAtMs = resolveEndTimestamp(context, metadata) ?? Date.now();
  const elapsedMs = Math.max(0, endAtMs - startedAtMs);

  if (timeoutMs) {
    return `${formatDuration(elapsedMs)} / ${formatDuration(timeoutMs)}`;
  }
  return formatDuration(elapsedMs);
}

function resolveEndTimestamp(
  context: ExecutionDetailsContext,
  metadata: CreateRepositorySandboxMetadata | undefined,
): number | undefined {
  if (metadata?.bootstrap?.finishedAt) {
    const ms = Date.parse(metadata.bootstrap.finishedAt);
    if (!Number.isNaN(ms)) {
      return ms;
    }
  }

  const finishedAt = (context.execution as { finishedAt?: string }).finishedAt;
  if (finishedAt) {
    const ms = Date.parse(finishedAt);
    if (!Number.isNaN(ms)) {
      return ms;
    }
  }

  return undefined;
}

function createRepositorySandboxMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const config = node.configuration as CreateRepositorySandboxConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.snapshot) {
    items.push({ icon: "container", label: config.snapshot });
  }

  if (config?.repository) {
    items.push({ icon: "git-branch", label: config.repository });
  }

  if (config?.bootstrap?.from) {
    items.push({ icon: "terminal", label: `bootstrap: ${config.bootstrap.from}` });
  }

  if (config?.bootstrap?.from === "file" && config?.bootstrap?.path) {
    items.push({ icon: "file-code", label: config.bootstrap.path });
  }

  return items;
}

function createRepositorySandboxSpecs(node: ComponentBaseContext["node"]): ComponentBaseSpec[] {
  const config = node.configuration as CreateRepositorySandboxConfiguration | undefined;
  const specs: ComponentBaseSpec[] = [];

  if (config?.bootstrap?.from === "inline" && config?.bootstrap?.script) {
    specs.push({ title: "Script", value: config.bootstrap.script });
  }

  return specs;
}
