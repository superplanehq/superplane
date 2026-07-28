import { afterEach, describe, expect, it } from "vitest";
import {
  clearAgentComposerSend,
  consumeAgentComposerSend,
  peekAgentComposerSend,
  requestAgentComposerSend,
} from "./composerPrefill";

const canvasA = "canvas-a";
const canvasB = "canvas-b";

afterEach(() => {
  clearAgentComposerSend();
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
});
