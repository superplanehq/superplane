import { describe, expect, it } from "bun:test";

import { appendSpokenPhrase, stripTrailingSpokenPhrase } from "./appendSpokenPhrase";

describe("appendSpokenPhrase", () => {
  it("appends the first phrase with no extra space", () => {
    expect(appendSpokenPhrase("", "Fix refunds")).toBe("Fix refunds");
  });

  it("inserts one space when the field already has text", () => {
    expect(appendSpokenPhrase("Fix refunds", "on retry")).toBe("Fix refunds on retry");
  });

  it("does not insert a space when the field already ends with whitespace", () => {
    expect(appendSpokenPhrase("Fix refunds ", "on retry")).toBe("Fix refunds on retry");
  });

  it("trims the spoken phrase", () => {
    expect(appendSpokenPhrase("Fix", "  refunds  ")).toBe("Fix refunds");
  });

  it("returns the current text when the phrase is empty", () => {
    expect(appendSpokenPhrase("Fix refunds", "   ")).toBe("Fix refunds");
  });

  it("truncates a title to 256 characters", () => {
    const current = "A".repeat(250);
    expect(appendSpokenPhrase(current, "1234567890", 256)).toBe(`${current} 12345`);
  });

  it("truncates a description to 5000 characters", () => {
    const current = "B".repeat(4996);
    expect(appendSpokenPhrase(current, "overflow", 5000)).toBe(`${current} ove`);
  });
});

describe("stripTrailingSpokenPhrase", () => {
  it("returns the value when the live phrase is empty", () => {
    expect(stripTrailingSpokenPhrase("Fix refunds", "  ")).toBe("Fix refunds");
  });

  it("clears the value when it is only the live phrase", () => {
    expect(stripTrailingSpokenPhrase("Fix refunds", "Fix refunds")).toBe("");
  });

  it("removes a spaced live suffix", () => {
    expect(stripTrailingSpokenPhrase("Please Fix refunds", "Fix refunds")).toBe("Please");
  });

  it("keeps the value when the live phrase is not a suffix", () => {
    expect(stripTrailingSpokenPhrase("Fix refunds now", "Fix refunds")).toBe("Fix refunds now");
  });
});
