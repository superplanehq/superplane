import { MetadataItem } from "@/ui/metadataList";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
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

interface CreateRepositorySandboxOutput {
  sandbox?: {
    id?: string;
    state?: string;
  };
  repository?: string;
  directory?: string;
  clone?: {
    exitCode?: number;
    result?: string;
  };
  bootstrap?: {
    from?: string;
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
    const details = baseMapper.getExecutionDetails(context);
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as CreateRepositorySandboxOutput | undefined;
    if (!data) {
      return details;
    }

    if (data.sandbox?.id) {
      details["Sandbox ID"] = data.sandbox.id;
    }
    if (data.sandbox?.state) {
      details["Sandbox State"] = data.sandbox.state;
    }
    if (data.repository) {
      details["Repository"] = data.repository;
    }
    if (data.directory) {
      details["Directory"] = data.directory;
    }
    if (typeof data.clone?.exitCode === "number") {
      details["Clone Exit Code"] = String(data.clone.exitCode);
    }
    if (typeof data.bootstrap?.exitCode === "number") {
      details["Bootstrap Exit Code"] = String(data.bootstrap.exitCode);
    }
    if (data.bootstrap?.from) {
      details["Bootstrap From"] = data.bootstrap.from;
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
