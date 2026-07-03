import { describe, expect, it } from "vitest";
import { resolveFitViewVersionId, shouldRefitOnInit, stampFittedContentKey } from "./fitView";

describe("resolveFitViewVersionId", () => {
  const liveParams = {
    liveCanvasVersionId: "live1",
    activeCanvasVersionId: "",
    isViewingDraftVersion: false,
    draftSpec: null as unknown,
    selectedVersion: null as { spec?: unknown } | null,
  };

  it("uses the live version id when no version is being previewed", () => {
    expect(resolveFitViewVersionId(liveParams)).toBe("live1");
  });

  it("keeps the live version id while a previewed version's spec is still loading (stale graph on screen)", () => {
    expect(resolveFitViewVersionId({ ...liveParams, activeCanvasVersionId: "v2", selectedVersion: {} })).toBe("live1");
  });

  it("switches to the previewed version id once its spec is loaded", () => {
    expect(resolveFitViewVersionId({ ...liveParams, activeCanvasVersionId: "v2", selectedVersion: { spec: {} } })).toBe(
      "v2",
    );
  });

  it("uses the draft version id only once the draft spec is available", () => {
    const draftLoading = { ...liveParams, activeCanvasVersionId: "d1", isViewingDraftVersion: true, draftSpec: null };
    expect(resolveFitViewVersionId(draftLoading)).toBe("live1");
    expect(resolveFitViewVersionId({ ...draftLoading, draftSpec: {} })).toBe("d1");
  });

  it("falls back to 'live' when there is no live version id", () => {
    expect(resolveFitViewVersionId({ ...liveParams, liveCanvasVersionId: undefined })).toBe("live");
  });
});

describe("shouldRefitOnInit", () => {
  it("fits on the first initialization", () => {
    expect(shouldRefitOnInit({ hasFittedBefore: false, fitViewContentKey: "c1:v1", lastFittedContentKey: null })).toBe(
      true,
    );
  });

  it("re-fits when the displayed content key changed", () => {
    expect(
      shouldRefitOnInit({ hasFittedBefore: true, fitViewContentKey: "c1:v2", lastFittedContentKey: "c1:v1" }),
    ).toBe(true);
  });

  it("restores instead of re-fitting when the content key is unchanged", () => {
    expect(
      shouldRefitOnInit({ hasFittedBefore: true, fitViewContentKey: "c1:v1", lastFittedContentKey: "c1:v1" }),
    ).toBe(false);
  });

  it("does not force a re-fit when there is no content key (run inspection)", () => {
    expect(
      shouldRefitOnInit({ hasFittedBefore: true, fitViewContentKey: undefined, lastFittedContentKey: "c1:v1" }),
    ).toBe(false);
  });
});

describe("stampFittedContentKey", () => {
  it("records the fitted content key", () => {
    const ref = { current: null as string | null };
    stampFittedContentKey(ref, "c1:v1");
    expect(ref.current).toBe("c1:v1");
  });

  it("ignores an undefined content key so a later init re-fits", () => {
    const ref = { current: "c1:v1" as string | null };
    stampFittedContentKey(ref, undefined);
    expect(ref.current).toBe("c1:v1");
  });

  it("is a no-op without a ref", () => {
    expect(() => stampFittedContentKey(undefined, "c1:v1")).not.toThrow();
  });
});
