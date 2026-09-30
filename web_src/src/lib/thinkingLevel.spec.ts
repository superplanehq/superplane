import { describe, expect, it } from "bun:test";

import { draftStartThinkingPayload } from "./thinkingLevel";

describe("draftStartThinkingPayload", () => {
  it("sends a concrete start level", () => {
    expect(draftStartThinkingPayload("medium")).toBe("medium");
    expect(draftStartThinkingPayload("low")).toBe("low");
    expect(draftStartThinkingPayload("high")).toBe("high");
  });

  it("omits an empty or auto value", () => {
    expect(draftStartThinkingPayload("")).toBeUndefined();
    expect(draftStartThinkingPayload("auto")).toBeUndefined();
  });

  it("keeps a stored default value", () => {
    expect(draftStartThinkingPayload("default")).toBe("default");
  });
});
