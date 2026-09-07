import { render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { FactoriesPaletteStoryShell } from "./factoriesPaletteStoryShell";

describe("FactoriesPaletteStoryShell", () => {
  afterEach(() => {
    document.body.classList.remove("theme-factories-palette-github");
  });

  it("does not add an overlay class for the Cursor palette", () => {
    render(
      <FactoriesPaletteStoryShell palette="cursor">
        <span>settings</span>
      </FactoriesPaletteStoryShell>,
    );

    expect(document.body.classList.contains("theme-factories-palette-github")).toBe(false);
  });

  it("adds and removes the GitHub overlay class on body", () => {
    const { unmount } = render(
      <FactoriesPaletteStoryShell palette="github">
        <span>settings</span>
      </FactoriesPaletteStoryShell>,
    );

    expect(document.body.classList.contains("theme-factories-palette-github")).toBe(true);
    unmount();
    expect(document.body.classList.contains("theme-factories-palette-github")).toBe(false);
  });
});
