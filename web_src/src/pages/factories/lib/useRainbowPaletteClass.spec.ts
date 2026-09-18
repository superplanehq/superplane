import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "bun:test";

import { useRainbowPaletteClass } from "./useRainbowPaletteClass";

const PALETTE_CLASS = "theme-factories-palette-rainbow";

describe("useRainbowPaletteClass", () => {
  afterEach(() => {
    document.body.classList.remove(PALETTE_CLASS);
  });

  it("adds the palette class to body on mount", () => {
    const { unmount } = renderHook(() => useRainbowPaletteClass());

    expect(document.body.classList.contains(PALETTE_CLASS)).toBe(true);
    unmount();
  });

  it("removes the palette class from body on unmount", () => {
    const { unmount } = renderHook(() => useRainbowPaletteClass());

    unmount();

    expect(document.body.classList.contains(PALETTE_CLASS)).toBe(false);
  });

  it("keeps the palette class until the last consumer unmounts", () => {
    const first = renderHook(() => useRainbowPaletteClass());
    const second = renderHook(() => useRainbowPaletteClass());

    first.unmount();
    expect(document.body.classList.contains(PALETTE_CLASS)).toBe(true);

    second.unmount();
    expect(document.body.classList.contains(PALETTE_CLASS)).toBe(false);
  });
});
