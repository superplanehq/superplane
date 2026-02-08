import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../types";
import { ComponentBaseProps, ComponentBaseSpec } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface GetPipelineConfiguration {
    pipelineId?: string;
}

export const getPipelineMapper: ComponentBaseMapper = {
    props(context: ComponentBaseContext): ComponentBaseProps {
        return {
            title:
                context.node.name ||
                context.componentDefinition.label ||
                context.componentDefinition.name ||
                "Unnamed component",
            iconSrc: SemaphoreLogo,
            iconSlug: context.componentDefinition.icon || "workflow",
            iconColor: getColorClass(context.componentDefinition?.color || "gray"),
            collapsed: context.node.isCollapsed,
            collapsedBackground: getBackgroundColorClass("white"),
            includeEmptyState: context.lastExecutions.length === 0,
            metadata: getPipelineMetadataList(context.node),
            specs: getPipelineSpecs(context.node),
        };
    },
    subtitle(context: SubtitleContext): string {
        const timestamp = context.execution.updatedAt || context.execution.createdAt;
        return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
    },
    getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
        const details: Record<string, any> = {};
        const outputs = context.execution.outputs as { default?: { data?: any }[] } | undefined;
        const payload = outputs?.default?.[0]?.data as Record<string, any> | undefined;

        if (!payload || typeof payload !== "object") {
            return details;
        }

        const addDetail = (key: string, value?: string) => {
            if (value) {
                details[key] = value;
            }
        };

        addDetail("Pipeline ID", payload.ppl_id);
        addDetail("Pipeline Name", payload.name);
        addDetail("Workflow ID", payload.wf_id);
        addDetail("State", payload.state);
        addDetail("Result", payload.result);

        return details;
    },
};

function getPipelineMetadataList(node: NodeInfo): MetadataItem[] {
    const metadata: MetadataItem[] = [];
    const configuration = node.configuration as GetPipelineConfiguration | undefined;

    if (configuration?.pipelineId) {
        metadata.push({ icon: "hash", label: configuration.pipelineId });
    }

    return metadata;
}

function getPipelineSpecs(_node: NodeInfo): ComponentBaseSpec[] {
    return [];
}
