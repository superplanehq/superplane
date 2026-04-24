import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import {
  addIfPresent,
  buildIncidentSelectionMetadata,
  formatDetailTimestamp,
  getFirstOutputData,
  getFirstOutputTimestamp,
  limitDetails,
  truncate,
  type Details,
} from "./incident_shared";
import type { AddIncidentActivityConfiguration, GrafanaIncidentActivity, GrafanaIncidentNodeMetadata } from "./types";

export const addIncidentActivityMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as AddIncidentActivityConfiguration | undefined;
    const nodeMetadata = context.node.metadata as GrafanaIncidentNodeMetadata | undefined;
    return grafanaComponentBaseProps(context, [
      ...buildIncidentSelectionMetadata(nodeMetadata, configuration?.incident),
      ...buildBodyMetadata(configuration?.body),
    ]);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Details {
    const activity = getFirstOutputData<GrafanaIncidentActivity>(context);
    const details: Details = {
      "Added At": formatDetailTimestamp(getFirstOutputTimestamp(context), context.execution.createdAt),
    };

    if (!activity) {
      return details;
    }

    addIfPresent(details, "Incident ID", activity.incidentID || activity.incidentId);
    addIfPresent(details, "Activity ID", activity.activityItemID || activity.activityId);
    addIfPresent(details, "Body", truncate(activity.body, 140));
    addIfPresent(details, "Created At", formatDetailTimestamp(activity.createdTime || activity.eventTime));
    addIfPresent(details, "Activity URL", activity.url);

    return limitDetails(details, 6);
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function buildBodyMetadata(body: string | undefined): MetadataItem[] {
  if (!body) {
    return [];
  }

  return [{ icon: "message-square", label: truncate(body, 70) }];
}
