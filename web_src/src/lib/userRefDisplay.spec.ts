import { describe, expect, it } from "vitest";
import { displayNameInitials, userRefDisplayProfile } from "./userRefDisplay";

describe("userRefDisplay", () => {
  it("derives initials from a display name", () => {
    expect(displayNameInitials("Alice Lovelace")).toBe("AL");
  });

  it("prefers organization directory data for avatars", () => {
    const directory = new Map([
      [
        "user-1",
        {
          name: "Ada Lovelace",
          initials: "AL",
          avatarUrl: "https://example.com/ada.png",
        },
      ],
    ]);

    expect(userRefDisplayProfile({ id: "user-1", name: "Ada" }, directory)).toEqual({
      name: "Ada Lovelace",
      initials: "AL",
      avatarUrl: "https://example.com/ada.png",
    });
  });

  it("falls back to owner name when directory data is unavailable", () => {
    expect(userRefDisplayProfile({ id: "user-2", name: "Bob Builder" })).toEqual({
      name: "Bob Builder",
      initials: "BB",
      avatarUrl: undefined,
    });
  });
});
