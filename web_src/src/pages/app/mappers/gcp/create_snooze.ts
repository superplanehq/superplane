import type { ComponentBaseProps } from "@/ui/componentBase";
import type { ComponentBaseContext, ComponentBaseMapper } from "../types";
import { baseProps, snoozeCreateMetadata, snoozeDetails, subtitle } from "./monitoring";

export const createSnoozeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "bell-off", "Create Snooze", snoozeCreateMetadata(context.node));
  },
  getExecutionDetails: snoozeDetails,
  subtitle,
};
