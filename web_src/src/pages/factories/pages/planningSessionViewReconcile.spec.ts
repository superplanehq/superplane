import { describe, expect, it } from "vitest";

import type { CreateWithAgentMessage } from "./createWithAgentTypes";
import { createWithAgentViewFromSession } from "./planningSessionView";

const base = { repository: "acme/payments", canvasId: "canvas-1", executionId: "exec-1" };

function rebuild(messages: unknown[], previousMessages: CreateWithAgentMessage[]) {
  return createWithAgentViewFromSession(
    { ...base, messages } as Parameters<typeof createWithAgentViewFromSession>[0],
    { composer: "", right: { kind: "empty" }, endConfirmOpen: false, previousMessages },
  );
}

function userMessage(id: string, iso: string): CreateWithAgentMessage {
  return { id, kind: "text", role: "user", text: "Retry please", createdAtMs: Date.parse(iso) };
}

describe("createWithAgentViewFromSession stale-snapshot reconciliation", () => {
  it("keeps an already-persisted message a stale snapshot omits", () => {
    const persisted = userMessage("msg-1", "2026-09-03T10:00:00Z");
    const local = userMessage("local-2", "2026-09-03T12:00:00Z");

    // A poll started before msg-1 persisted arrives late and lists no messages.
    const view = rebuild([], [persisted, local]);

    // The stale snapshot must not drop msg-1, and the pending bubble stays.
    expect(view.messages.map((message) => message.id)).toEqual(["msg-1", "local-2"]);
  });

  it("does not retire a pending bubble after a stale snapshot re-lists an old identical message", () => {
    // Reproduces the flicker race: two identical sends, the first persisted as
    // msg-1, the second still pending as local-9. An out-of-order snapshot
    // interleaves before the second send persists.
    const msg1 = { id: "msg-1", role: "user", text: "Retry please", createdAt: "2026-09-03T10:00:00Z" };
    const afterSecondSend = [userMessage("msg-1", "2026-09-03T10:00:00Z"), userMessage("local-9", "2026-09-03T12:00:00Z")];

    // A stale snapshot (predates msg-1) omits it entirely.
    const afterStale = rebuild([], afterSecondSend);
    expect(afterStale.messages.map((message) => message.id)).toEqual(["msg-1", "local-9"]);

    // A current snapshot that still only lists msg-1 must not retire local-9.
    const afterCurrent = rebuild([msg1], afterStale.messages);
    expect(afterCurrent.messages.map((message) => message.id)).toEqual(["msg-1", "local-9"]);

    // Once the second send persists, the bubble retires with no duplicate.
    const msg2 = { id: "msg-2", role: "user", text: "Retry please", createdAt: "2026-09-03T12:00:00Z" };
    const afterPersist = rebuild([msg1, msg2], afterCurrent.messages);
    expect(afterPersist.messages.map((message) => message.id)).toEqual(["msg-1", "msg-2"]);
    expect(afterPersist.messages.some((message) => message.id.startsWith("local-"))).toBe(false);
  });
});
