import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { executionSubtitle, propsWithMetadata, readDefaultOutput, projectMetadata, type ProjectRefConfiguration } from "./shared";
import { stringOrDash } from "./common";

function nameMetadata(label: string, field: string): (node: NodeInfo) => MetadataItem[] {
  return (node) => {
    const configuration = node.configuration as Record<string, string | undefined> | undefined;
    const metadata: MetadataItem[] = [];
    if (configuration?.[field]) {
      metadata.push({ icon: "tag", label: `${label}: ${configuration[field]}` });
    }
    return metadata;
  };
}

interface ProjectOutput {
  projectId?: string;
  name?: string;
  framework?: string;
}

interface EnvVarConfiguration extends ProjectRefConfiguration {
  key?: string;
  targets?: string[];
}

interface EnvVarOutput {
  projectId?: string;
  key?: string;
  target?: string[];
  envId?: string;
}

interface DomainConfiguration extends ProjectRefConfiguration {
  domain?: string;
}

interface DomainOutput {
  projectId?: string;
  name?: string;
  verified?: boolean;
  removed?: boolean;
}

function envVarMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as EnvVarConfiguration | undefined;
  const metadata: MetadataItem[] = [...projectMetadata(node)];

  if (configuration?.key) {
    metadata.push({ icon: "key", label: `Key: ${configuration.key}` });
  }
  if (configuration?.targets && configuration.targets.length > 0) {
    metadata.push({ icon: "flag", label: `Environments: ${configuration.targets.join(", ")}` });
  }

  return metadata;
}

function projectDetailsMapper(
  metadata: (node: NodeInfo) => MetadataItem[],
  timestampLabel: string,
): ComponentBaseMapper {
  return {
    props: (context) => propsWithMetadata(context, metadata),

    subtitle: executionSubtitle,

    getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
      const output = readDefaultOutput(context);
      const result = output?.data as ProjectOutput | undefined;

      return {
        "Project ID": stringOrDash(result?.projectId),
        Name: stringOrDash(result?.name),
        Framework: stringOrDash(result?.framework),
        [timestampLabel]: context.execution.createdAt
          ? new Date(context.execution.createdAt).toLocaleString()
          : "-",
      };
    },
  };
}

export const getProjectMapper: ComponentBaseMapper = projectDetailsMapper(projectMetadata, "Retrieved At");

export const createProjectMapper: ComponentBaseMapper = projectDetailsMapper(
  nameMetadata("Name", "name"),
  "Created At",
);

export const upsertEnvVarMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, envVarMetadata),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = readDefaultOutput(context);
    const result = output?.data as EnvVarOutput | undefined;

    return {
      Key: stringOrDash(result?.key),
      Environments: stringOrDash(result?.target?.join(", ")),
      "Env ID": stringOrDash(result?.envId),
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };
  },
};

function domainMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as DomainConfiguration | undefined;
  const metadata: MetadataItem[] = [...projectMetadata(node)];

  if (configuration?.domain) {
    metadata.push({ icon: "globe", label: `Domain: ${configuration.domain}` });
  }

  return metadata;
}

export const addDomainMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, domainMetadata),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = readDefaultOutput(context);
    const result = output?.data as DomainOutput | undefined;

    const details: Record<string, string> = {
      Domain: stringOrDash(result?.name),
      Verified: stringOrDash(result?.verified),
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    return details;
  },
};

export const removeDomainMapper: ComponentBaseMapper = {
  props: (context) => propsWithMetadata(context, domainMetadata),

  subtitle: executionSubtitle,

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = readDefaultOutput(context);
    const result = output?.data as DomainOutput | undefined;

    return {
      Domain: stringOrDash(result?.name),
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };
  },
};
