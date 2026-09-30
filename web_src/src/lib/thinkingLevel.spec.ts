import { describe, expect, it } from "bun:test";

import { draftStartThinkingPayload, modelNameWithThinking, visibleThinkingLevelLabel } from "./thinkingLevel";

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

describe("visibleThinkingLevelLabel", () => {
  it("shows Low, Medium, High, and a stored Default", () => {
    expect(visibleThinkingLevelLabel("low")).toBe("Low");
    expect(visibleThinkingLevelLabel("medium")).toBe("Medium");
    expect(visibleThinkingLevelLabel("high")).toBe("High");
    expect(visibleThinkingLevelLabel("default")).toBe("Default");
  });

  it("hides Auto and an empty value", () => {
    expect(visibleThinkingLevelLabel("auto")).toBeUndefined();
    expect(visibleThinkingLevelLabel("")).toBeUndefined();
    expect(visibleThinkingLevelLabel(undefined)).toBeUndefined();
  });
});

describe("modelNameWithThinking", () => {
  it("puts the thinking word after the model name", () => {
    expect(modelNameWithThinking("opus 4-6", "medium")).toBe("opus 4-6 Medium");
  });

  it("leaves the model name alone for Auto", () => {
    expect(modelNameWithThinking("opus 4-6", "auto")).toBe("opus 4-6");
  });

  it("adds no word when the model name is empty", () => {
    expect(modelNameWithThinking("", "medium")).toBe("");
  });
});
