import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { Comment } from "./types";

export const createIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName =
      context.componentDefinition.name || context.node.componentName || context.componentDefinition.label || "unknown";

    if (lastExecution?.rootEvent) {
      const rootTriggerNode = context.nodes.find((node) => node.id === lastExecution.rootEvent?.nodeId);
      const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
      const { title: eventTitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: lastExecution.rootEvent });
      const eventSection: EventSection = {
        receivedAt: new Date(lastExecution.createdAt!),
        eventTitle: eventTitle,
        eventSubtitle: buildGithubExecutionSubtitle(lastExecution),
        eventState: getState(componentName)(lastExecution),
        eventId: lastExecution.rootEvent.id!,
      };
      props.eventSections = [eventSection];
    }

    return props;
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const comment = outputs.default[0].data as Comment;
      Object.assign(details, {
        "Created At": comment.created_at ? new Date(comment.created_at).toLocaleString() : "-",
      });

      details["Comment URL"] = comment.html_url || "";
    }

    return details;
  },
};
