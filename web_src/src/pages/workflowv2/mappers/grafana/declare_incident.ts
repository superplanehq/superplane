import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import { buildIncidentDetails, getFirstOutputData, truncate, type Details } from "./incident_shared";
import type { DeclareIncidentConfiguration, GrafanaIncident } from "./types";

export const declareIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as DeclareIncidentConfiguration | undefined;
    return grafanaComponentBaseProps(context, buildDeclareIncidentMetadata(configuration));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Details {
    const incident = getFirstOutputData<GrafanaIncident>(context);
    return buildIncidentDetails(context, incident, "Declared At", [
      "Title",
      "Severity",
      "Status",
      "Labels",
      "Incident URL",
    ]);
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function buildDeclareIncidentMetadata(configuration: DeclareIncidentConfiguration | undefined): MetadataItem[] {
  const metadata: MetadataItem[] = [];

  if (configuration?.title) {
    metadata.push({ icon: "alert-triangle", label: truncate(configuration.title, 60) });
  }
  if (configuration?.severity) {
    metadata.push({ icon: "funnel", label: `Severity: ${configuration.severity}` });
  }
  if (configuration?.isDrill) {
    metadata.push({ icon: "shield-alert", label: "Drill" });
  }

  return metadata;
}
