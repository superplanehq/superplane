import { describe, expect, it } from "vitest";
import {
  formatWorkOrderIdentifier,
  isValidWorkspaceKey,
  isWorkspaceKeyShaped,
  normalizeWorkspaceKey,
  suggestWorkspaceKeyFromName,
  uniqueWorkspaceKeyFromName,
} from "./workspaceKey";

describe("normalizeWorkspaceKey", () => {
  it("lowercases and strips non-letters", () => {
    expect(normalizeWorkspaceKey("SP-1")).toBe("sp");
    expect(normalizeWorkspaceKey(" OP S ")).toBe("ops");
  });

  it("caps to the max length so users cannot exceed it", () => {
    expect(normalizeWorkspaceKey("superplane")).toBe("super");
  });
});

describe("isValidWorkspaceKey", () => {
  it.each([
    ["", false],
    ["a", false],
    ["abcdef", false],
    ["AB", false],
    ["sp", true],
    ["super", true],
    ["sp2", false],
  ])("returns %s for %s", (input, expected) => {
    expect(isValidWorkspaceKey(input as string)).toBe(expected);
  });
});

describe("isWorkspaceKeyShaped", () => {
  it("accepts both cases so legacy uppercase keys still match", () => {
    expect(isWorkspaceKeyShaped("SP")).toBe(true);
    expect(isWorkspaceKeyShaped("sp")).toBe(true);
    expect(isWorkspaceKeyShaped("Sp")).toBe(true);
  });

  it("rejects the wrong length or non-letters", () => {
    expect(isWorkspaceKeyShaped("a")).toBe(false);
    expect(isWorkspaceKeyShaped("abcdef")).toBe(false);
    expect(isWorkspaceKeyShaped("sp2")).toBe(false);
  });
});

describe("suggestWorkspaceKeyFromName", () => {
  it("returns the leading letters trimmed to the max length", () => {
    expect(suggestWorkspaceKeyFromName("SuperPlane")).toBe("super");
    expect(suggestWorkspaceKeyFromName("Ops team")).toBe("opste");
  });

  it("returns an empty string when the name has fewer than two letters", () => {
    expect(suggestWorkspaceKeyFromName("12345")).toBe("");
    expect(suggestWorkspaceKeyFromName("A!")).toBe("");
  });
});

describe("uniqueWorkspaceKeyFromName", () => {
  it("returns the derived key when it is free", () => {
    expect(uniqueWorkspaceKeyFromName("Payments Service", [])).toBe("payme");
  });

  it("walks the last letter through the alphabet on collision", () => {
    expect(uniqueWorkspaceKeyFromName("Payments Service", ["payme"])).toBe("payma");
    expect(uniqueWorkspaceKeyFromName("Payments Service", ["payme", "payma"])).toBe("paymb");
  });

  it("falls back to the default seed when the name has no usable letters", () => {
    expect(uniqueWorkspaceKeyFromName("12345", [])).toBe("ws");
  });

  it("matches taken keys case-insensitively", () => {
    expect(uniqueWorkspaceKeyFromName("Payments Service", ["PAYME"])).toBe("payma");
  });
});

describe("formatWorkOrderIdentifier", () => {
  it("formats a valid key and number", () => {
    expect(formatWorkOrderIdentifier("sp", 42)).toBe("sp-42");
  });

  it("returns an empty string when either value is missing", () => {
    expect(formatWorkOrderIdentifier("", 1)).toBe("");
    expect(formatWorkOrderIdentifier("sp", 0)).toBe("");
    expect(formatWorkOrderIdentifier("sp", null)).toBe("");
    expect(formatWorkOrderIdentifier(null, 1)).toBe("");
  });
});
