import { CanvasesCanvasNodeExecution } from "@/api-client";
import {
  ComponentBaseProps,
  ComponentBaseSpec,
  DEFAULT_EVENT_STATE_MAP,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGitlabExecutionSubtitle } from "./utils";
import { MetadataItem } from "@/ui/metadataList";

interface PipelineMetadata {
  id?: number;
  iid?: number;
  status?: string;
  url?: string;
}

interface ExecutionMetadata {
  pipeline?: PipelineMetadata;
}

interface RunPipelineConfiguration {
  project: string;
  ref: string;
  inputs: Array<{ name: string; value: string }>;
}

export const RUN_PIPELINE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  passed: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const runPipelineStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  const outputs = execution.outputs as { passed?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
  if (outputs?.failed && outputs.failed.length > 0) {
    return "failed";
  }
  if (outputs?.passed && outputs.passed.length > 0) {
    return "passed";
  }

  const metadata = execution.metadata as ExecutionMetadata;
  switch (metadata?.pipeline?.status) {
    case "success":
      return "passed";
    case "failed":
    case "canceled":
    case "cancelled":
    case "skipped":
    case "manual":
    case "blocked":
      return "failed";
    default:
      return "neutral";
  }
};

export const RUN_PIPELINE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUN_PIPELINE_STATE_MAP,
  getState: runPipelineStateFunction,
};

export const runPipelineMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const config = context.node.configuration as RunPipelineConfiguration;
    const metadata = base.metadata as MetadataItem[];
    if (config.ref) {
      metadata.push({ icon: "git-branch", label: config.ref });
    }

    return {
      ...base,
      specs: runPipelineSpecs(context.node.configuration),
      eventStateMap: RUN_PIPELINE_STATE_MAP,
      metadata: metadata,
    };
  },

  subtitle(context: SubtitleContext): string {
    const metadata = context.execution.metadata as ExecutionMetadata | undefined;
    const status = metadata?.pipeline?.status ? metadata.pipeline.status : "Pipeline Run";
    return buildGitlabExecutionSubtitle(context.execution, status);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const metadata = context.execution.metadata as ExecutionMetadata | undefined;
    const details: Record<string, string> = {};
    const pipeline = metadata?.pipeline;

    if (pipeline?.id) {
      details["ID"] = pipeline.id.toString();
    }
    if (pipeline?.iid) {
      details["IID"] = pipeline.iid.toString();
    }
    if (pipeline?.status) {
      details["Status"] = pipeline.status;
    }
    if (pipeline?.url) {
      details["URL"] = pipeline.url;
    }
    if (context.execution.createdAt) {
      details["Started At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    if (context.execution.updatedAt) {
      details["Last Updated At"] = new Date(context.execution.updatedAt).toLocaleString();
    }

    return details;
  },
};

function runPipelineSpecs(configuration: unknown): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const config = configuration as RunPipelineConfiguration;
  const inputs = config?.inputs;

  if (inputs && inputs.length > 0) {
    specs.push({
      title: "input",
      tooltipTitle: "pipeline inputs",
      iconSlug: "settings",
      values: inputs.map((input) => ({
        badges: [
          {
            label: input.name,
            bgColor: "bg-indigo-100",
            textColor: "text-indigo-800",
          },
          {
            label: input.value,
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
        ],
      })),
    });
  }

  return specs;
}
