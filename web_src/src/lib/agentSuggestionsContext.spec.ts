import { afterEach, describe, expect, it } from "vitest";
import {
  AGENT_SUGGESTIONS_STORAGE_KEY,
  clearAgentSuggestions,
  getAgentSuggestions,
  setAgentSuggestions,
} from "./agentSuggestionsContext";

const canvasId = "canvas-1";

afterEach(() => {
  sessionStorage.clear();
});

describe("agentSuggestionsContext", () => {
  it("stores and returns suggestions for a canvas", () => {
    setAgentSuggestions(canvasId, [{ id: "add-ci", label: "Add CI", prompt: "Add CI to this canvas" }]);

    expect(getAgentSuggestions(canvasId)).toEqual([{ id: "add-ci", label: "Add CI", prompt: "Add CI to this canvas" }]);
    expect(getAgentSuggestions("other-canvas")).toEqual([]);
  });

  it("clears suggestions for a single canvas", () => {
    setAgentSuggestions(canvasId, [{ id: "a", label: "A", prompt: "A" }]);
    setAgentSuggestions("other", [{ id: "b", label: "B", prompt: "B" }]);

    clearAgentSuggestions(canvasId);

    expect(getAgentSuggestions(canvasId)).toEqual([]);
    expect(getAgentSuggestions("other")).toEqual([{ id: "b", label: "B", prompt: "B" }]);
  });

  it("ignores corrupt session storage payloads", () => {
    sessionStorage.setItem(AGENT_SUGGESTIONS_STORAGE_KEY, "{not-json");
    expect(getAgentSuggestions(canvasId)).toEqual([]);
  });
});
