import { afterEach, describe, expect, it } from "vitest";
import {
  clearAgentComposerSend,
  consumeAgentComposerSend,
  peekAgentComposerSend,
  requestAgentComposerSend,
  requeueAgentComposerSend,
} from "./composerPrefill";

afterEach(() => {
  clearAgentComposerSend();
});

describe("composerPrefill", () => {
  it("stores a pending send until consumed", () => {
    requestAgentComposerSend("  Add CI  ");
    expect(peekAgentComposerSend()).toBe("Add CI");
    expect(consumeAgentComposerSend()).toBe("Add CI");
    expect(peekAgentComposerSend()).toBeNull();
  });

  it("queues multiple pending sends in order", () => {
    requestAgentComposerSend("first");
    requestAgentComposerSend("second");
    expect(consumeAgentComposerSend()).toBe("first");
    expect(consumeAgentComposerSend()).toBe("second");
    expect(consumeAgentComposerSend()).toBeNull();
  });

  it("ignores blank requests", () => {
    requestAgentComposerSend("   ");
    expect(peekAgentComposerSend()).toBeNull();
  });

  it("requeues a failed send at the front of the queue", () => {
    requestAgentComposerSend("second");
    requeueAgentComposerSend("first");
    expect(consumeAgentComposerSend()).toBe("first");
    expect(consumeAgentComposerSend()).toBe("second");
  });
});
