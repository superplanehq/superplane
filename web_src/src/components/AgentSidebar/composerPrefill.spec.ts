import { afterEach, describe, expect, it, vi } from "vitest";
import {
  clearAgentComposerPrefill,
  clearAgentComposerSend,
  consumeAgentComposerPrefill,
  consumeAgentComposerSend,
  peekAgentComposerSend,
  requestAgentComposerPrefill,
  requestAgentComposerSend,
} from "./composerPrefill";

const canvasA = "canvas-a";
const canvasB = "canvas-b";

afterEach(() => {
  clearAgentComposerSend();
  clearAgentComposerPrefill();
  vi.restoreAllMocks();
});

describe("composerPrefill", () => {
  it("stores a pending send until consumed", () => {
    requestAgentComposerSend(canvasA, "  Add CI  ");
    expect(peekAgentComposerSend(canvasA)).toBe("Add CI");
    expect(consumeAgentComposerSend(canvasA)).toBe("Add CI");
    expect(peekAgentComposerSend(canvasA)).toBeNull();
  });

  it("queues multiple pending sends in order for a canvas", () => {
    requestAgentComposerSend(canvasA, "first");
    requestAgentComposerSend(canvasA, "second");
    expect(consumeAgentComposerSend(canvasA)).toBe("first");
    expect(consumeAgentComposerSend(canvasA)).toBe("second");
    expect(consumeAgentComposerSend(canvasA)).toBeNull();
  });

  it("keeps pending sends isolated per canvas", () => {
    requestAgentComposerSend(canvasA, "for-a");
    requestAgentComposerSend(canvasB, "for-b");

    expect(peekAgentComposerSend(canvasA)).toBe("for-a");
    expect(peekAgentComposerSend(canvasB)).toBe("for-b");
    expect(consumeAgentComposerSend(canvasB)).toBe("for-b");
    expect(peekAgentComposerSend(canvasA)).toBe("for-a");
  });

  it("ignores blank requests and missing canvas ids", () => {
    requestAgentComposerSend(canvasA, "   ");
    requestAgentComposerSend("", "hello");
    expect(peekAgentComposerSend(canvasA)).toBeNull();
  });

  it("keeps node context as a separate prefill from the auto-send queue", () => {
    requestAgentComposerPrefill(canvasA, {
      text: "How can I improve @Deploy?",
      mentions: [{ type: "node", id: "deploy", label: "Deploy" }],
    });

    expect(consumeAgentComposerPrefill(canvasA)).toEqual({
      text: "How can I improve @Deploy?",
      mentions: [{ type: "node", id: "deploy", label: "Deploy" }],
    });
    expect(peekAgentComposerSend(canvasA)).toBeNull();
  });

  it("does not inject stale node context later", () => {
    const now = vi.spyOn(Date, "now").mockReturnValueOnce(1_000).mockReturnValue(11_001);
    requestAgentComposerPrefill(canvasA, {
      text: "How can I improve @Deploy?",
      mentions: [{ type: "node", id: "deploy", label: "Deploy" }],
    });

    expect(consumeAgentComposerPrefill(canvasA)).toBeNull();
    now.mockRestore();
  });
});
