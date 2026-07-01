import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext } from "../types";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import {
  buildIncidentDetails,
  buildIncidentSelectionMetadata,
  getFirstOutputData,
  type Details,
} from "./incident_shared";
import type { GrafanaIncident, GrafanaIncidentNodeMetadata, IncidentSelectionConfiguration } from "./types";

export const getIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as IncidentSelectionConfiguration | undefined;
    const nodeMetadata = context.node.metadata as GrafanaIncidentNodeMetadata | undefined;
    return grafanaComponentBaseProps(context, buildIncidentSelectionMetadata(nodeMetadata, configuration?.incident));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Details {
    const incident = getFirstOutputData<GrafanaIncident>(context);
    return buildIncidentDetails(context, incident, "Fetched At", [
      "Title",
      "Severity",
      "Status",
      "Labels",
      "Incident URL",
    ]);
  },

  subtitle: grafanaCreatedAtSubtitle,
};
