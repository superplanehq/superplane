import { useState } from "react";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  CustomFieldRenderer,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import {
  ComponentBaseProps,
  ComponentBaseSpec,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";
import jenkinsIcon from "@/assets/icons/integrations/jenkins.svg";
import { formatTimeAgo } from "@/utils/date";
import { CanvasesCanvasNodeExecution } from "@/api-client";

interface NodeMetadata {
  job?: { name: string; url: string };
  webhookUrl?: string;
}
interface ExecutionMetadata {
  job?: {
    name: string;
    url: string;
  };
  build?: {
    number: number;
    url: string;
    result: string;
    building: boolean;
  };
}

export const TRIGGER_BUILD_STATE_MAP: EventStateMap = {
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
  stopped: {
    icon: "circle-stop",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
};

export const triggerBuildStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
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

  const metadata = execution.metadata as ExecutionMetadata;
  const buildResult = metadata?.build?.result;
  if (buildResult === "FAILURE" || buildResult === "UNSTABLE" || buildResult === "ABORTED") {
    return "failed";
  }

  return "passed";
};

export const TRIGGER_BUILD_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TRIGGER_BUILD_STATE_MAP,
  getState: triggerBuildStateFunction,
};

export const triggerBuildMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: jenkinsIcon,
      iconSlug: context.componentDefinition.icon || "jenkins",
      iconColor: getColorClass(context.componentDefinition?.color || "gray"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: lastExecution ? triggerBuildEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: triggerBuildMetadataList(context.node),
      specs: triggerBuildSpecs(context.node),
      eventStateMap: TRIGGER_BUILD_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = context.execution.outputs as
      | { passed?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
      | undefined;
    const payload =
      (outputs?.passed?.[0]?.data as Record<string, any> | undefined) ||
      (outputs?.failed?.[0]?.data as Record<string, any> | undefined) ||
      (outputs?.default?.[0]?.data as Record<string, any> | undefined);
    const payloadData =
      payload && typeof payload === "object" && payload.data && typeof payload.data === "object"
        ? payload.data
        : payload;

    if (!payloadData || typeof payloadData !== "object") {
      return details;
    }

    const build = payloadData.build as Record<string, any> | undefined;
    const job = payloadData.job as Record<string, any> | undefined;

    const addDetail = (key: string, value?: string) => {
      if (value) {
        details[key] = value;
      }
    };

    addDetail("Job", job?.name);
    addDetail("Build URL", build?.url);
    addDetail("Build Number", build?.number?.toString());
    addDetail("Result", build?.result);
    addDetail("Duration", build?.duration ? `${Math.round(build.duration / 1000)}s` : undefined);

    return details;
  },
};

function triggerBuildMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;
  const nodeMetadata = node.metadata as any;

  if (nodeMetadata?.job?.name) {
    metadata.push({ icon: "folder", label: nodeMetadata.job.name });
  } else if (configuration?.job) {
    metadata.push({ icon: "folder", label: configuration.job });
  }

  return metadata;
}

function triggerBuildSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as any;

  const parameters = configuration?.parameters as Array<{ name: string; value: string }> | undefined;
  if (parameters && parameters.length > 0) {
    specs.push({
      title: "parameter",
      tooltipTitle: "build parameters",
      iconSlug: "settings",
      values: parameters.map((param) => ({
        badges: [
          {
            label: param.name || "",
            bgColor: "bg-purple-100",
            textColor: "text-purple-800",
          },
          {
            label: param.value || "",
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
        ],
      })),
    });
  }

  return specs;
}

const CopyCodeButton: React.FC<{ code: string }> = ({ code }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_err) {
      showErrorToast("Failed to copy text");
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1 bg-white outline-1 outline-black/20 hover:outline-black/30 rounded text-gray-600 dark:text-gray-400"
      title={copied ? "Copied!" : "Copy to clipboard"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
    </button>
  );
};

export const triggerBuildCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as NodeMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl;

    if (!webhookUrl) {
      return (
        <div className="border-t-1 border-gray-200 pt-4">
          <p className="text-sm text-gray-500 dark:text-gray-400">Save the canvas to generate the webhook URL.</p>
        </div>
      );
    }

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Jenkins Notification Plugin</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Configure the Jenkins Notification Plugin to POST build events to this URL.
            </p>
            <div className="mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Webhook URL
              </label>
              <div className="relative group mt-1">
                <input
                  type="text"
                  value={webhookUrl}
                  readOnly
                  className="w-full text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono"
                />
                <CopyCodeButton code={webhookUrl} />
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};

function triggerBuildEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const sections: EventSection[] = [];

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const executionState = triggerBuildStateFunction(execution);
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined;

  sections.push({
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle,
    eventState: executionState,
    eventId: execution.rootEvent!.id!,
  });

  return sections;
}
