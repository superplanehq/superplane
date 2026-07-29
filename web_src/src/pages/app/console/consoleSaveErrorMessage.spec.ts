import { describe, expect, it } from "vitest";

import { formatConsoleSaveErrorMessage } from "./consoleSaveErrorMessage";

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
