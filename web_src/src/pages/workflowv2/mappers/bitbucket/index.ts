import { TriggerRenderer } from "../types";
import { onPushTriggerRenderer } from "./on_push";

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
};
