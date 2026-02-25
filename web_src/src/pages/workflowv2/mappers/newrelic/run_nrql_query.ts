import { ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext, ComponentBaseContext, ExecutionInfo, NodeInfo } from "../types";
import { OutputPayload } from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";
import newrelicIcon from "@/assets/icons/integrations/newrelic.svg";

/**
 * Mapper for the "newrelic.runNRQLQuery" component.
 * Provides custom execution details showing query results and metadata.
 */
export const runNrqlQueryMapper: ComponentBaseMapper = {
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

        if (data?.query) {
            details["Query"] = data.query;
        }

        if (data?.accountId) {
            details["Account ID"] = data.accountId;
        }

        if (payload?.timestamp) {
            details["Executed At"] = new Date(payload.timestamp).toLocaleString();
        }

        if (data?.results?.length != null) {
            details["Result Count"] = String(data.results.length);
        }

        if (data?.metadata?.timeWindow) {
            const { begin, end } = data.metadata.timeWindow;
            if (begin && end) {
                details["Time Window"] = `${new Date(begin).toLocaleString()} — ${new Date(end).toLocaleString()}`;
            }
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
