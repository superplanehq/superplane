import type { ComponentBaseProps } from "@/ui/componentBase";
import type { ComponentBaseContext, ComponentBaseMapper } from "../types";
import { baseProps, snoozeDetails, snoozeSelectorMetadata, subtitle } from "./monitoring";

export const expireSnoozeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "bell-off", "Expire Snooze", snoozeSelectorMetadata(context.node));
  },
  getExecutionDetails: snoozeDetails,
  subtitle,
};
