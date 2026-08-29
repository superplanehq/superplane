import { describe, expect, it } from "vitest";

import { getNameInitials } from "./nameInitials";

describe("getNameInitials", () => {
  it("drops emoji-only words and keeps the first + last letter word", () => {
    expect(getNameInitials("SuperPlane Prod 🚀")).toBe("SP");
  });

  it("returns empty for an emoji-only name", () => {
    expect(getNameInitials("🚀")).toBe("");
    expect(getNameInitials("🚀 🎉")).toBe("");
  });

  it("returns the single letter for a one-word name", () => {
    expect(getNameInitials("SuperPlane")).toBe("S");
  });

  it("returns empty for an empty name", () => {
    expect(getNameInitials("")).toBe("");
  });

  it("returns empty for a whitespace-only name", () => {
    expect(getNameInitials("   ")).toBe("");
  });

  it("takes the first letter of the first and last word for two words", () => {
    expect(getNameInitials("Alex Reviewer")).toBe("AR");
  });

  it("keeps a word made only of a digit", () => {
    expect(getNameInitials("Prod 2")).toBe("P2");
  });

  it("takes the first + last word for three or more words", () => {
    expect(getNameInitials("Mary Jane Watson")).toBe("MW");
  });
});
