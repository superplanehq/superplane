import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesWorkOrderEvent } from "@/api-client";

const { showInfoToastMock } = vi.hoisted(() => ({ showInfoToastMock: vi.fn() }));

vi.mock("@/lib/toast", () => ({
  showInfoToast: showInfoToastMock,
}));

import { useWorkOrderMentionToast } from "./useWorkOrderMentionToast";

const CURRENT_USER_ID = "user-me";
const AUTHOR_ID = "user-author";

function commentEvent(body: string, authorId: string, mentionIds: string[], timestamp: string): FactoriesWorkOrderEvent {
  return {
    type: "order.comment.added",
    timestamp,
    event: {
      body,
      author: { kind: "user", userId: authorId },
      mentions: mentionIds.map((id) => ({ id })),
    },
  };
}

const resolveUserName = (userId: string | undefined) => (userId === AUTHOR_ID ? "Alex Reviewer" : userId);

describe("useWorkOrderMentionToast", () => {
  beforeEach(() => {
    showInfoToastMock.mockClear();
  });

  it("does not toast for mentions already present on first render", () => {
    const events = [commentEvent("hey @[Me](user:x)", AUTHOR_ID, [CURRENT_USER_ID], "t1")];
    renderHook(({ events }) => useWorkOrderMentionToast(events, CURRENT_USER_ID, resolveUserName, "Work order #1"), {
      initialProps: { events },
    });

    expect(showInfoToastMock).not.toHaveBeenCalled();
  });

  it("toasts when a new comment mentioning the current user arrives", () => {
    const initialEvents = [commentEvent("first", AUTHOR_ID, [], "t1")];
    const { rerender } = renderHook(
      ({ events }) => useWorkOrderMentionToast(events, CURRENT_USER_ID, resolveUserName, "Work order #1"),
      { initialProps: { events: initialEvents } },
    );

    expect(showInfoToastMock).not.toHaveBeenCalled();

    const nextEvents = [...initialEvents, commentEvent("take a look @[Me](user:x)", AUTHOR_ID, [CURRENT_USER_ID], "t2")];
    rerender({ events: nextEvents });

    expect(showInfoToastMock).toHaveBeenCalledWith("Alex Reviewer mentioned you in Work order #1.");
  });

  it("does not toast for a self-mention", () => {
    const initialEvents: FactoriesWorkOrderEvent[] = [];
    const { rerender } = renderHook(
      ({ events }) => useWorkOrderMentionToast(events, CURRENT_USER_ID, resolveUserName, "Work order #1"),
      { initialProps: { events: initialEvents } },
    );

    const nextEvents = [commentEvent("noting this myself", CURRENT_USER_ID, [CURRENT_USER_ID], "t1")];
    rerender({ events: nextEvents });

    expect(showInfoToastMock).not.toHaveBeenCalled();
  });

  it("does not toast for a new comment that doesn't mention the current user", () => {
    const initialEvents: FactoriesWorkOrderEvent[] = [];
    const { rerender } = renderHook(
      ({ events }) => useWorkOrderMentionToast(events, CURRENT_USER_ID, resolveUserName, "Work order #1"),
      { initialProps: { events: initialEvents } },
    );

    const nextEvents = [commentEvent("no mentions", AUTHOR_ID, [], "t1")];
    rerender({ events: nextEvents });

    expect(showInfoToastMock).not.toHaveBeenCalled();
  });
});
