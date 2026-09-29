import { describe, expect, it } from "bun:test";

import type { EventInfo, TriggerEventContext } from "../types";
import { onMessageTriggerRenderer } from "./on_message";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-01-01T00:00:00Z").toISOString(),
    nodeId: "node-1",
    type: "gcp.pubsub.onMessage",
    data,
  };
}

describe("onMessageTriggerRenderer", () => {
  it("getTitleAndSubtitle uses messageId in the title", () => {
    const context: TriggerEventContext = { event: event({ messageId: "abcdef123456" }) };
    expect(onMessageTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Received Pub/Sub message · abcdef12");
  });

  it("getTitleAndSubtitle is a bare title when messageId is absent", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onMessageTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Received Pub/Sub message");
  });

  it("getRootEventValues maps messageId and publishTime", () => {
    const publishTime = "2026-01-01T12:00:00Z";
    const context: TriggerEventContext = {
      event: event({ messageId: "msg-1", publishTime }),
    };
    const values = onMessageTriggerRenderer.getRootEventValues(context);
    expect(values["Message ID"]).toBe("msg-1");
    expect(values["Published At"]).toBe(new Date(publishTime).toLocaleString());
    expect(values["Received At"]).toBe(new Date("2026-01-01T00:00:00Z").toLocaleString());
  });
});
