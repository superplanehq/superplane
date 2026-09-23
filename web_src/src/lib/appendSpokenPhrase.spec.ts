import { describe, expect, it } from "bun:test";

import { appendSpokenPhrase } from "./appendSpokenPhrase";

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
