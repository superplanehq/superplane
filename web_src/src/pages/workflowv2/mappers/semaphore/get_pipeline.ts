import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
    ComponentBaseContext,
    ComponentBaseMapper,
    ExecutionDetailsContext,
    ExecutionInfo,
    NodeInfo,
    OutputPayload,
    SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import semaphoreIcon from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface SemaphorePipeline {
    name?: string;
    ppl_id?: string;
    wf_id?: string;
    state?: string;
    result?: string;
}

export const getPipelineMapper: ComponentBaseMapper = {
    props(context: ComponentBaseContext): ComponentBaseProps {
        const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
        const componentName = context.componentDefinition.name || "unknown";

        return {
            iconSrc: semaphoreIcon,
            collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
            collapsed: context.node.isCollapsed,
            title:
                context.node.name ||
                context.componentDefinition.label ||
                context.componentDefinition.name ||
                "Unnamed component",
            eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
            metadata: metadataList(context.node),
            includeEmptyState: !lastExecution,
            eventStateMap: getStateMap(componentName),
        };
    },

    getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
        const outputs = context.execution.outputs as { default: OutputPayload[] };
        if (!outputs?.default || outputs.default.length === 0) {
            return {};
        }
        const pipeline = outputs.default[0].data as SemaphorePipeline;
        return getDetailsForPipeline(pipeline);
    },

    subtitle(context: SubtitleContext): string {
        if (!context.execution.createdAt) return "";
        return formatTimeAgo(new Date(context.execution.createdAt));
    },
};

function getDetailsForPipeline(pipeline: SemaphorePipeline): Record<string, string> {
    const details: Record<string, string> = {};

    details["Pipeline Name"] = pipeline?.name || "-";
    details["Pipeline ID"] = pipeline?.ppl_id || "-";
    details["Workflow ID"] = pipeline?.wf_id || "-";
    details["State"] = pipeline?.state || "-";
    details["Result"] = pipeline?.result || "-";

    return details;
}

function metadataList(node: NodeInfo): MetadataItem[] {
    const metadata: MetadataItem[] = [];
    const configuration = node.configuration as { pipelineId?: string };

    if (configuration?.pipelineId) {
        metadata.push({ icon: "search", label: "Pipeline: " + configuration.pipelineId });
    }

    return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
    const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

    return [
        {
            receivedAt: new Date(execution.createdAt!),
            eventTitle: title,
            eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
            eventState: getState(componentName)(execution),
            eventId: execution.rootEvent!.id!,
        },
    ];
}
