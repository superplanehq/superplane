import { describe, expect, it } from "vitest";
import { GENERIC_FAILURE_MESSAGE, sanitizeErrorMessage } from "./agentChatUi";

describe("sanitizeErrorMessage", () => {
  it("returns the message for a clean Error", () => {
    expect(sanitizeErrorMessage(new Error("Rate limit reached. Please wait a moment and try again."))).toBe(
      "Rate limit reached. Please wait a moment and try again.",
    );
  });

  it("returns generic message for non-Error values", () => {
    expect(sanitizeErrorMessage("some string")).toBe(GENERIC_FAILURE_MESSAGE);
    expect(sanitizeErrorMessage(null)).toBe(GENERIC_FAILURE_MESSAGE);
    expect(sanitizeErrorMessage(undefined)).toBe(GENERIC_FAILURE_MESSAGE);
    expect(sanitizeErrorMessage(42)).toBe(GENERIC_FAILURE_MESSAGE);
  });

  it("rejects messages containing braces (raw exception reprs)", () => {
    const rawError = new Error(
      "{'type': 'error', 'error': {'details': None, 'type': 'overloaded_error', 'message': 'Overloaded'}}",
    );
    expect(sanitizeErrorMessage(rawError)).toBe(GENERIC_FAILURE_MESSAGE);
  });

  it("rejects messages containing Traceback", () => {
    expect(sanitizeErrorMessage(new Error("Traceback (most recent call last):"))).toBe(GENERIC_FAILURE_MESSAGE);
  });

  it("rejects messages exceeding 200 characters", () => {
    const longMessage = "a".repeat(201);
    expect(sanitizeErrorMessage(new Error(longMessage))).toBe(GENERIC_FAILURE_MESSAGE);
  });

  it("accepts messages at exactly 200 characters", () => {
    const exactMessage = "a".repeat(200);
    expect(sanitizeErrorMessage(new Error(exactMessage))).toBe(exactMessage);
  });

  it("rejects empty error messages", () => {
    expect(sanitizeErrorMessage(new Error(""))).toBe(GENERIC_FAILURE_MESSAGE);
  });
});
