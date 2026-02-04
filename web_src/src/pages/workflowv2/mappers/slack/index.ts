import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onAppMentionTriggerRenderer } from "./on_app_mention";
import { sendTextMessageMapper } from "./send_text_message";
import { sendAndWaitMessageMapper } from "./send_and_wait_message";
import { buildActionStateRegistry } from "../utils";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { defaultStateFunction } from "../stateRegistry";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendTextMessage: sendTextMessageMapper,
  sendAndWaitMessage: sendAndWaitMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAppMention: onAppMentionTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendTextMessage: buildActionStateRegistry("sent"),
  sendAndWaitMessage: {
    stateMap: {
      ...DEFAULT_EVENT_STATE_MAP,
      received: {
        ...DEFAULT_EVENT_STATE_MAP.success,
        label: "RECEIVED",
      },
      timeout: {
        ...DEFAULT_EVENT_STATE_MAP.neutral,
        label: "TIMEOUT",
      },
      waiting: {
        icon: "clock",
        textColor: "text-gray-800",
        backgroundColor: "bg-orange-100",
        badgeColor: "bg-yellow-600",
        label: "WAITING",
      },
    },
    getState: (execution) => {
      if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
        return "waiting";
      }

      const state = defaultStateFunction(execution);
      if (state !== "success") return state;

      const outputs = execution.outputs as { received?: unknown[]; timeout?: unknown[] } | undefined;
      if (outputs?.timeout && outputs.timeout.length > 0) {
        return "timeout";
      }
      return "received";
    },
  },
};
