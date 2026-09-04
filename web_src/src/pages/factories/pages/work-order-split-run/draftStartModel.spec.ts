import { describe, expect, it } from "vitest";

import {
  DRAFT_START_MODEL_AUTO,
  displayRunnerModel,
  draftStartModelPayload,
  phaseWithRunnerModel,
  runnerModelsFromCanvasNodes,
} from "./draftStartModel";

describe("draftStartModelPayload", () => {
  it("sends no model for Auto", () => {
    expect(draftStartModelPayload(DRAFT_START_MODEL_AUTO)).toBeUndefined();
    expect(draftStartModelPayload("")).toBeUndefined();
    expect(draftStartModelPayload("  ")).toBeUndefined();
  });

  it("sends the listed id", () => {
    expect(draftStartModelPayload("claude-opus-4-6")).toBe("claude-opus-4-6");
  });
});

describe("displayRunnerModel", () => {
  it("keeps a short alias", () => {
    expect(displayRunnerModel("opus")).toBe("opus");
  });

  it("keeps the last path segment of an OpenRouter id", () => {
    expect(displayRunnerModel("anthropic/claude-opus-4-6")).toBe("claude-opus-4-6");
  });
});

describe("runnerModelsFromCanvasNodes", () => {
  it("joins distinct runner models from canvas nodes", () => {
    expect(
      runnerModelsFromCanvasNodes([
        { configuration: { model: "anthropic/claude-opus-4-6" } },
        { configuration: { model: "anthropic/claude-opus-4-6" } },
        { configuration: { model: "anthropic/claude-sonnet-4-6" } },
      ]),
    ).toBe("anthropic/claude-opus-4-6 · anthropic/claude-sonnet-4-6");
  });
});

describe("phaseWithRunnerModel", () => {
  it("keeps a model already on the phase", () => {
    expect(phaseWithRunnerModel({ model: "opus" }, [{ configuration: { model: "sonnet" } }]).model).toBe("opus");
  });

  it("fills Auto from the canvas when the phase has no model", () => {
    expect(phaseWithRunnerModel({}, [{ configuration: { model: "opus" } }]).model).toBe("opus");
  });

  it("keeps a start model the canvas runners can use", () => {
    expect(
      phaseWithRunnerModel({ model: "claude-opus-4-6" }, [
        { component: "runnerClaudeCode", configuration: { model: "claude-sonnet-4-6" } },
      ]).model,
    ).toBe("claude-opus-4-6");
  });

  it("uses the canvas model when the start model is for another runner", () => {
    expect(
      phaseWithRunnerModel({ model: "claude-opus-4-6" }, [
        { component: "runnerCodex", configuration: { model: "gpt-5" } },
      ]).model,
    ).toBe("gpt-5");
  });
});
