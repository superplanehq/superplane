import { describe, expect, it } from "vitest";

import {
  isDraftCanvasLoadingWhileEditing,
  isEditBootstrapReady,
  isEditStagingActionsReady,
} from "./edit-staging-ready";

describe("isEditBootstrapReady", () => {
  it("waits for baselines and staged draft before edit UI is shown", () => {
    expect(
      isEditBootstrapReady({
        isEditing: true,
        isEnteringEditSession: false,
        stagingBaselinesReady: false,
        draftCanvasSpec: { nodes: [] },
        shouldReadStagedCanvasVersion: true,
      }),
    ).toBe(false);

    expect(
      isEditBootstrapReady({
        isEditing: true,
        isEnteringEditSession: false,
        stagingBaselinesReady: true,
        draftCanvasSpec: null,
        shouldReadStagedCanvasVersion: true,
      }),
    ).toBe(false);

    expect(
      isEditBootstrapReady({
        isEditing: true,
        isEnteringEditSession: false,
        stagingBaselinesReady: true,
        draftCanvasSpec: { nodes: [] },
        shouldReadStagedCanvasVersion: true,
      }),
    ).toBe(true);
  });
});

describe("isEditStagingActionsReady", () => {
  it("is always ready outside edit mode", () => {
    expect(
      isEditStagingActionsReady({
        isEditing: false,
        isEnteringEditSession: true,
        stagingBaselinesReady: false,
      }),
    ).toBe(true);
  });

  it("waits for the edit-session bootstrap and committed baselines", () => {
    expect(
      isEditStagingActionsReady({
        isEditing: true,
        isEnteringEditSession: true,
        stagingBaselinesReady: false,
      }),
    ).toBe(false);

    expect(
      isEditStagingActionsReady({
        isEditing: true,
        isEnteringEditSession: false,
        stagingBaselinesReady: false,
      }),
    ).toBe(false);

    expect(
      isEditStagingActionsReady({
        isEditing: true,
        isEnteringEditSession: false,
        stagingBaselinesReady: true,
        draftCanvasSpec: { nodes: [] },
        shouldReadStagedCanvasVersion: true,
      }),
    ).toBe(true);
  });
});

describe("isDraftCanvasLoadingWhileEditing", () => {
  it("keeps loading until edit bootstrap is fully ready", () => {
    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: true,
        shouldReadStagedCanvasVersion: true,
        isEnteringEditSession: false,
        isEditBootstrapReady: false,
        draftCanvasSpec: { nodes: [] },
        loadedCanvasVersionLoading: false,
        loadedCanvasVersionFetching: false,
      }),
    ).toBe(true);

    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: true,
        shouldReadStagedCanvasVersion: true,
        isEnteringEditSession: false,
        isEditBootstrapReady: true,
        draftCanvasSpec: { nodes: [] },
        loadedCanvasVersionLoading: false,
        loadedCanvasVersionFetching: false,
      }),
    ).toBe(false);
  });

  it("shows loading while entering edit mode before the edit session is active", () => {
    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: false,
        shouldReadStagedCanvasVersion: false,
        isEnteringEditSession: true,
        isEditBootstrapReady: false,
        draftCanvasSpec: null,
        loadedCanvasVersionLoading: false,
        loadedCanvasVersionFetching: false,
      }),
    ).toBe(true);
  });

  it("does not load outside edit mode when not entering edit", () => {
    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: false,
        shouldReadStagedCanvasVersion: true,
        isEnteringEditSession: false,
        isEditBootstrapReady: true,
        draftCanvasSpec: null,
        loadedCanvasVersionLoading: true,
        loadedCanvasVersionFetching: false,
      }),
    ).toBe(false);

    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: true,
        shouldReadStagedCanvasVersion: false,
        isEnteringEditSession: false,
        isEditBootstrapReady: true,
        draftCanvasSpec: null,
        loadedCanvasVersionLoading: true,
        loadedCanvasVersionFetching: false,
      }),
    ).toBe(false);
  });

  it("shows loading while entering edit mode before staged draft is applied", () => {
    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: true,
        shouldReadStagedCanvasVersion: true,
        isEnteringEditSession: true,
        isEditBootstrapReady: false,
        draftCanvasSpec: null,
        loadedCanvasVersionLoading: false,
        loadedCanvasVersionFetching: false,
      }),
    ).toBe(true);
  });

  it("stops loading once the staged draft spec is available", () => {
    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: true,
        shouldReadStagedCanvasVersion: true,
        isEnteringEditSession: false,
        isEditBootstrapReady: true,
        draftCanvasSpec: { nodes: [] },
        loadedCanvasVersionLoading: true,
        loadedCanvasVersionFetching: true,
      }),
    ).toBe(false);
  });

  it("keeps loading until staged version data arrives when draft spec is still empty", () => {
    expect(
      isDraftCanvasLoadingWhileEditing({
        isEditing: true,
        shouldReadStagedCanvasVersion: true,
        isEnteringEditSession: false,
        isEditBootstrapReady: false,
        draftCanvasSpec: null,
        loadedCanvasVersionLoading: true,
        loadedCanvasVersionFetching: false,
      }),
    ).toBe(true);
  });
});
