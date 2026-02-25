import { ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext, ComponentBaseContext, ExecutionInfo, NodeInfo } from "../types";
import { OutputPayload } from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";
import newrelicIcon from "@/assets/icons/integrations/newrelic.svg";

/**
 * Mapper for the "newrelic.reportMetric" component.
 * Provides execution details showing metric submission results.
 */
export const reportMetricMapper: ComponentBaseMapper = {
    props(context: ComponentBaseContext): ComponentBaseProps {
        const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
        const componentName = context.componentDefinition.name || "unknown";

        return {
            title:
                context.node.name ||
                context.componentDefinition.label ||
                context.componentDefinition.name ||
                "Unnamed component",
            iconSrc: newrelicIcon,
            iconSlug: "newrelic",
            iconColor: getColorClass(context.componentDefinition.color),
            collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
            collapsed: context.node.isCollapsed,
            eventSections: lastExecution ? getEventSections(context.nodes, lastExecution, componentName) : undefined,
            includeEmptyState: !lastExecution,
            eventStateMap: getStateMap(componentName),
        };
    },

    subtitle(context: SubtitleContext): string {
        const timestamp = context.execution.updatedAt || context.execution.createdAt;
        return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
    },

    getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
        const details: Record<string, any> = {};
        const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
        const payload = outputs?.default?.[0];

        if (!payload?.data) return details;

        const data = payload.data as any;

        if (data?.statusCode) {
            details["HTTP Status"] = String(data.statusCode);
        }

        if (data?.status) {
            details["Status"] = data.status;
        }

        if (payload?.timestamp) {
            details["Reported At"] = new Date(payload.timestamp).toLocaleString();
        }

        return details;
    },
};

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
    const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
    const subtitleTimestamp = execution.updatedAt || execution.createdAt;
    const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

    return [
        {
            receivedAt: new Date(execution.createdAt!),
            eventTitle: title,
            eventSubtitle,
            eventState: getState(componentName)(execution),
            eventId: execution.rootEvent?.id || "",
        },
    ];
}
