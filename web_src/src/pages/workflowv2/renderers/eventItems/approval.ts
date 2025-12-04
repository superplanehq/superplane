import { EventItemStyleProps, EventItemRenderer } from "./types";
import { getDefaultEventItemStyle } from "./default";

export const approvalEventItemRenderer: EventItemRenderer = {
  getEventItemStyle: (state: string, componentType?: string, eventData?: Record<string, unknown>): EventItemStyleProps => {
    // For approval components, check if this is an approval state first
    if (componentType === "approval") {
      // Check for approval-specific states first
      if (eventData) {
        const approvalState = eventData.state as string;

        if (approvalState === "approved") {
          return {
            iconName: "check",
            iconColor: "text-green-700",
            iconBackground: "bg-green-200",
            titleColor: "text-green-800",
            iconSize: 16,
            iconContainerSize: 4,
            iconStrokeWidth: 2,
            animation: "",
          };
        }

        if (approvalState === "rejected") {
          return {
            iconName: "x",
            iconColor: "text-red-700",
            iconBackground: "bg-red-200",
            titleColor: "text-red-800",
            iconSize: 16,
            iconContainerSize: 4,
            iconStrokeWidth: 2,
            animation: "",
          };
        }

        if (approvalState === "error") {
          return {
            iconName: "triangle-alert",
            iconColor: "text-red-700",
            iconBackground: "bg-red-200",
            titleColor: "text-red-800",
            iconSize: 16,
            iconContainerSize: 4,
            iconStrokeWidth: 2,
            animation: "",
          };
        }
      }

      // For approval components, also check if we can map standard states
      if (state === "processed") {
        return {
          iconName: "check",
          iconColor: "text-green-700",
          iconBackground: "bg-green-200",
          titleColor: "text-green-800",
          iconSize: 16,
          iconContainerSize: 4,
          iconStrokeWidth: 2,
          animation: "",
        };
      }

      if (state === "discarded") {
        return {
          iconName: "x",
          iconColor: "text-red-700",
          iconBackground: "bg-red-200",
          titleColor: "text-red-800",
          iconSize: 16,
          iconContainerSize: 4,
          iconStrokeWidth: 2,
          animation: "",
        };
      }
    }

    // Fall back to default styling
    return getDefaultEventItemStyle(state);
  },
};