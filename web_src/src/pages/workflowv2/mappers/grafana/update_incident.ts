import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import {
  buildIncidentDetails,
  buildIncidentSelectionMetadata,
  getFirstOutputData,
  type Details,
} from "./incident_shared";
import type { GrafanaIncident, GrafanaIncidentNodeMetadata, UpdateIncidentConfiguration } from "./types";

export const updateIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as UpdateIncidentConfiguration | undefined;
    const nodeMetadata = context.node.metadata as GrafanaIncidentNodeMetadata | undefined;
    return grafanaComponentBaseProps(context, [
      ...buildIncidentSelectionMetadata(nodeMetadata, configuration?.incident),
      ...buildUpdatedFieldsMetadata(configuration),
    ]);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Details {
    const incident = getFirstOutputData<GrafanaIncident>(context);
    return buildIncidentDetails(context, incident, "Updated At", [
      "Title",
      "Severity",
      "Status",
      "Labels",
      "Incident URL",
    ]);
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function buildUpdatedFieldsMetadata(configuration: UpdateIncidentConfiguration | undefined): MetadataItem[] {
  const fields: string[] = [];
  if (configuration?.title) fields.push("Title");
  if (configuration?.severity) fields.push("Severity");
  if (Array.isArray(configuration?.labels) && configuration.labels.length > 0)
    fields.push(`Labels (${configuration.labels.length})`);
  if (typeof configuration?.isDrill === "boolean") fields.push("Drill");

  if (fields.length === 0) {
    return [];
  }

  return [{ icon: "settings", label: `Updating: ${fields.join(", ")}` }];
}
