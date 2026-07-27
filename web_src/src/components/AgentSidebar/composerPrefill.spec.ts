import { afterEach, describe, expect, it } from "vitest";
import { consumeAgentComposerSend, peekAgentComposerSend, requestAgentComposerSend } from "./composerPrefill";

afterEach(() => {
  consumeAgentComposerSend();
});

describe("composerPrefill", () => {
  it("stores a pending send until consumed", () => {
    requestAgentComposerSend("  Add CI  ");
    expect(peekAgentComposerSend()).toBe("Add CI");
    expect(consumeAgentComposerSend()).toBe("Add CI");
    expect(peekAgentComposerSend()).toBeNull();
  });

  it("ignores blank requests", () => {
    requestAgentComposerSend("   ");
    expect(peekAgentComposerSend()).toBeNull();
  });
});
