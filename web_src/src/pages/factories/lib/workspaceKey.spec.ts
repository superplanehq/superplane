import { describe, expect, it } from "vitest";
import {
  formatWorkOrderIdentifier,
  isValidWorkspaceKey,
  normalizeWorkspaceKey,
  suggestWorkspaceKeyFromName,
} from "./workspaceKey";

describe("normalizeWorkspaceKey", () => {
  it("uppercases and strips non-letters", () => {
    expect(normalizeWorkspaceKey("sp-1")).toBe("SP");
    expect(normalizeWorkspaceKey(" op s ")).toBe("OPS");
  });

  it("caps to the max length so users cannot exceed it", () => {
    expect(normalizeWorkspaceKey("SUPERPLANE")).toBe("SUPER");
  });
});

describe("isValidWorkspaceKey", () => {
  it.each([
    ["", false],
    ["A", false],
    ["ABCDEF", false],
    ["ab", false],
    ["SP", true],
    ["SUPER", true],
    ["SP2", false],
  ])("returns %s for %s", (input, expected) => {
    expect(isValidWorkspaceKey(input as string)).toBe(expected);
  });
});

describe("suggestWorkspaceKeyFromName", () => {
  it("returns the leading letters trimmed to the max length", () => {
    expect(suggestWorkspaceKeyFromName("SuperPlane")).toBe("SUPER");
    expect(suggestWorkspaceKeyFromName("Ops team")).toBe("OPSTE");
  });

  it("returns an empty string when the name has fewer than two letters", () => {
    expect(suggestWorkspaceKeyFromName("12345")).toBe("");
    expect(suggestWorkspaceKeyFromName("A!")).toBe("");
  });
});

describe("formatWorkOrderIdentifier", () => {
  it("formats a valid key and number", () => {
    expect(formatWorkOrderIdentifier("SP", 42)).toBe("SP-42");
  });

  it("returns an empty string when either value is missing", () => {
    expect(formatWorkOrderIdentifier("", 1)).toBe("");
    expect(formatWorkOrderIdentifier("SP", 0)).toBe("");
    expect(formatWorkOrderIdentifier("SP", null)).toBe("");
    expect(formatWorkOrderIdentifier(null, 1)).toBe("");
  });
});
