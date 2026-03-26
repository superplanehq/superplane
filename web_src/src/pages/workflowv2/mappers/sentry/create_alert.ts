import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  buildEventSections,
  executionDetailsForSentryMetricAlertRule,
  getAlertRuleProjectLabel,
  getAlertThresholdMetadataLabel,
  subtitleForSentryMetricAlertRule,
  type AlertRuleNodeMetadata,
  type SentryAlertThresholdConfiguration,
} from "./utils";
import type { ComponentBaseContext, ComponentBaseMapper, NodeInfo } from "../types";

interface CreateAlertConfiguration {
  project?: string;
  name?: string;
  environment?: string;
  critical?: SentryAlertThresholdConfiguration;
  warning?: SentryAlertThresholdConfiguration;
}

export const createAlertMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: sentryIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution
        ? buildEventSections(context.nodes, lastExecution, componentName, getTriggerRenderer, getState)
        : undefined,
      metadata: buildMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle: subtitleForSentryMetricAlertRule,

  getExecutionDetails: executionDetailsForSentryMetricAlertRule,
};

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as CreateAlertConfiguration | undefined;
  const nodeMetadata = node.metadata as AlertRuleNodeMetadata | undefined;
  const metadata = [];

  const projectLabel = getAlertRuleProjectLabel(nodeMetadata, configuration);
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }
  if (configuration?.name) {
    metadata.push({ icon: "siren", label: configuration.name });
  }
  const thresholdLabel = getAlertThresholdMetadataLabel(configuration);
  if (thresholdLabel) {
    metadata.push({ icon: "triangle-alert", label: thresholdLabel });
  }

  return metadata.slice(0, 3);
}
