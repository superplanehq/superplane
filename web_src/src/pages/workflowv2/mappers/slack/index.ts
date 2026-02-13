import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onAppMentionTriggerRenderer } from "./on_app_mention";
import { sendTextMessageMapper } from "./send_text_message";
import { sendAndWaitMapper } from "./send_and_wait";
import { buildActionStateRegistry } from "../utils";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { defaultStateFunction } from "../stateRegistry";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendTextMessage: sendTextMessageMapper,
  sendAndWait: sendAndWaitMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAppMention: onAppMentionTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendTextMessage: buildActionStateRegistry("sent"),
  sendAndWait: {
    stateMap: {
      ...DEFAULT_EVENT_STATE_MAP,
      waiting: {
        icon: "refresh-cw",
        textColor: "text-gray-800",
        backgroundColor: "bg-sky-100",
        badgeColor: "bg-blue-500",
        label: "Waiting",
      },
      received: DEFAULT_EVENT_STATE_MAP.success,
      timed_out: {
        icon: "clock",
        textColor: "text-gray-800",
        backgroundColor: "bg-orange-100",
        badgeColor: "bg-orange-500",
        label: "Timed Out",
      },
    },
    getState: (execution) => {
      const metadata = execution.metadata as { state?: string } | undefined;
      if (metadata?.state === "waiting") return "waiting" as any;
      if (metadata?.state === "timed_out") return "timed_out" as any;
      if (metadata?.state === "received") return "received" as any;
      return defaultStateFunction(execution);
    },
  },
};
