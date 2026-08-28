import { describe, expect, it } from "vitest";
import { FACTORY_LAYOUT_ANIMATION_DURATION_MS } from "./nodePositionAnimation";
import {
  FACTORY_CONFIGURE_FIT_SETTLE_MS,
  factoryConfigureEnterFitViewOptions,
  shouldFitFactoryConfigureEnter,
} from "./factoryConfigureFitView";

describe("shouldFitFactoryConfigureEnter", () => {
  const ready = {
    factoryConfigure: true,
    isEditing: true,
    hasReactFlowInitialized: true,
    hasFittedThisVisit: false,
    nodeCount: 2,
  };

  it("fits once when Configure edit is ready and nodes exist", () => {
    expect(shouldFitFactoryConfigureEnter(ready)).toBe(true);
  });

  it("does not fit in view mode", () => {
    expect(shouldFitFactoryConfigureEnter({ ...ready, factoryConfigure: false })).toBe(false);
  });

  it("does not fit before the edit session is active", () => {
    expect(shouldFitFactoryConfigureEnter({ ...ready, isEditing: false })).toBe(false);
  });

  it("does not fit before React Flow init", () => {
    expect(shouldFitFactoryConfigureEnter({ ...ready, hasReactFlowInitialized: false })).toBe(false);
  });

  it("does not fit again in the same Configure visit", () => {
    expect(shouldFitFactoryConfigureEnter({ ...ready, hasFittedThisVisit: true })).toBe(false);
  });

  it("does not fit an empty canvas", () => {
    expect(shouldFitFactoryConfigureEnter({ ...ready, nodeCount: 0 })).toBe(false);
  });
});

describe("FACTORY_CONFIGURE_FIT_SETTLE_MS", () => {
  it("waits for the factory layout animation plus a short measure buffer", () => {
    expect(FACTORY_CONFIGURE_FIT_SETTLE_MS).toBe(FACTORY_LAYOUT_ANIMATION_DURATION_MS + 50);
  });
});

describe("factoryConfigureEnterFitViewOptions", () => {
  it("frames the whole graph at 100% zoom", () => {
    expect(factoryConfigureEnterFitViewOptions()).toEqual({
      includeHiddenNodes: true,
      minZoom: 1,
      maxZoom: 1,
      padding: 0.08,
      duration: 500,
    });
  });

  it("centers a deep-linked node at 100% zoom", () => {
    const focusNode = { id: "generate-pr-text" };
    expect(factoryConfigureEnterFitViewOptions(focusNode)).toEqual({
      includeHiddenNodes: true,
      minZoom: 1,
      maxZoom: 1,
      padding: 0.08,
      duration: 500,
      nodes: [focusNode],
    });
  });
});
