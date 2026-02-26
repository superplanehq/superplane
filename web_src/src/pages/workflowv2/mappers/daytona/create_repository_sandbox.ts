import { MetadataItem } from "@/ui/metadataList";
import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper } from "./base";

interface CreateRepositorySandboxConfiguration {
  snapshot?: string;
  target?: string;
  repository?: string;
  bootstrap?: {
    from?: string;
    script?: string;
    path?: string;
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
  };
}

export const createRepositorySandboxMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    return {
      ...props,
      metadata: createRepositorySandboxMetadataList(context.node),
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

    return details;
  },
};

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

  if (config?.bootstrap?.path) {
    items.push({ icon: "file-code", label: config.bootstrap.path });
  }

  return items;
}
