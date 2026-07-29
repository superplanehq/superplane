import { describe, expect, it } from "vitest";

import { consoleSaveErrorReason, formatConsoleSaveErrorMessage } from "./consoleSaveErrorMessage";

describe("consoleSaveErrorReason", () => {
  it("strips the mutation's 'invalid console yaml: ' wrapper prefix", () => {
    // Callers use this helper to prefix the reason with their own
    // surface-specific copy ("Failed to save console: …" for the
    // overlay, "Could not save console.yaml: …" for the Files tab).
    // Stripping the wrapper prefix keeps the reason surface-agnostic.
    const error = new Error(`invalid console yaml: Page "main": Too many panels (max 20 per page).`);
    expect(consoleSaveErrorReason(error)).toBe(`Page "main": Too many panels (max 20 per page).`);
  });

  it("passes non-wrapped errors through verbatim", () => {
    expect(consoleSaveErrorReason(new Error("Network request failed"))).toBe("Network request failed");
  });

  it("stringifies non-Error thrown values", () => {
    expect(consoleSaveErrorReason("boom")).toBe("boom");
  });

  it("falls back to 'unknown error' when the message is empty", () => {
    expect(consoleSaveErrorReason(new Error(""))).toBe("unknown error");
    expect(consoleSaveErrorReason(new Error("invalid console yaml: "))).toBe("unknown error");
  });
});

describe("formatConsoleSaveErrorMessage", () => {
  it("strips the mutation's 'invalid console yaml: ' wrapper prefix", () => {
    // The mutation wraps validation failures with this prefix. Users
    // only need the underlying reason, which is what
    // `validateConsolePagesDelta` and `validateConsolePagesStructural`
    // return.
    const error = new Error(`invalid console yaml: Page "main": Too many panels (max 20 per page).`);
    expect(formatConsoleSaveErrorMessage(error)).toBe(
      `Failed to save console: Page "main": Too many panels (max 20 per page).`,
    );
  });

  it("passes non-wrapped errors through verbatim", () => {
    // Network / server errors should surface as-is so operators can
    // still triage them from the toast.
    const error = new Error("Network request failed");
    expect(formatConsoleSaveErrorMessage(error)).toBe("Failed to save console: Network request failed");
  });

  it("stringifies non-Error thrown values", () => {
    expect(formatConsoleSaveErrorMessage("boom")).toBe("Failed to save console: boom");
    expect(formatConsoleSaveErrorMessage({ toString: () => "custom" })).toBe("Failed to save console: custom");
  });

  it("falls back to 'unknown error' when the message is empty", () => {
    // A wrapper prefix without a detail (e.g., "invalid console
    // yaml: ") would otherwise produce a dangling ": " suffix. This
    // also guards against `new Error("")` and equivalents.
    expect(formatConsoleSaveErrorMessage(new Error(""))).toBe("Failed to save console: unknown error");
    expect(formatConsoleSaveErrorMessage(new Error("invalid console yaml: "))).toBe(
      "Failed to save console: unknown error",
    );
  });
});
