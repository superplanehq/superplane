import { describe, expect, it } from "vitest";

import { initialsForName } from "./factoriesRail";

describe("initialsForName", () => {
  it("falls back to '?' when no word has a letter or digit", () => {
    expect(initialsForName("")).toBe("?");
    expect(initialsForName("🚀")).toBe("?");
  });

  it("drops emoji-only words and keeps the first + last letter word", () => {
    expect(initialsForName("SuperPlane Prod 🚀")).toBe("SP");
  });
});
