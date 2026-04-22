import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import {
  buildIncidentDetails,
  buildIncidentSelectionMetadata,
  getFirstOutputData,
  truncate,
  type Details,
} from "./incident_shared";
import type { GrafanaIncident, GrafanaIncidentNodeMetadata, ResolveIncidentConfiguration } from "./types";

export const resolveIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as ResolveIncidentConfiguration | undefined;
    const nodeMetadata = context.node.metadata as GrafanaIncidentNodeMetadata | undefined;
    return grafanaComponentBaseProps(context, [
      ...buildIncidentSelectionMetadata(nodeMetadata, configuration?.incident),
      ...buildSummaryMetadata(configuration?.summary),
    ]);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Details {
    const incident = getFirstOutputData<GrafanaIncident>(context);
    return buildIncidentDetails(context, incident, "Resolved At", [
      "Title",
      "Severity",
      "Status",
      "Labels",
      "Incident URL",
    ]);
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function buildSummaryMetadata(summary: string | undefined): MetadataItem[] {
  if (!summary) {
    return [];
  }

  return [{ icon: "message-square", label: `Summary: ${truncate(summary, 60)}` }];
}
