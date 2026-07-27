import { afterEach, describe, expect, it } from "vitest";
import {
  AGENT_SUGGESTIONS_STORAGE_KEY,
  clearAgentSuggestions,
  dismissAgentSuggestion,
  getAgentSuggestions,
  getDismissedAgentSuggestionIds,
  setAgentSuggestions,
} from "./agentSuggestionsContext";

const canvasId = "canvas-1";

afterEach(() => {
  sessionStorage.clear();
  localStorage.clear();
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

  it("persists dismissed suggestion ids per canvas", () => {
    dismissAgentSuggestion(canvasId, "add-ci");
    dismissAgentSuggestion(canvasId, "add-ci");
    dismissAgentSuggestion("other", "slack");

    expect(Array.from(getDismissedAgentSuggestionIds(canvasId))).toEqual(["add-ci"]);
    expect(Array.from(getDismissedAgentSuggestionIds("other"))).toEqual(["slack"]);
  });

  it("ignores corrupt session storage payloads", () => {
    sessionStorage.setItem(AGENT_SUGGESTIONS_STORAGE_KEY, "{not-json");
    expect(getAgentSuggestions(canvasId)).toEqual([]);
  });
});
