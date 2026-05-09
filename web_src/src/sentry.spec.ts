import { describe, expect, it } from "vitest";
import { IGNORED_ERROR_PATTERNS } from "./sentry";

const matchesAny = (message: string) =>
  IGNORED_ERROR_PATTERNS.some((pattern) =>
    typeof pattern === "string" ? message.includes(pattern) : pattern.test(message),
  );

describe("Sentry IGNORED_ERROR_PATTERNS", () => {
  it("matches the Telegram WebApp postEvent noise reported by browser extensions", () => {
    expect(matchesAny("Error invoking postEvent: Method not found")).toBe(true);
    expect(matchesAny("Error: Error invoking postEvent: Method not found")).toBe(true);
    expect(matchesAny("postEvent: Some other error: Method not found")).toBe(true);
  });

  it("does not match unrelated application errors", () => {
    expect(matchesAny("TypeError: Cannot read properties of undefined")).toBe(false);
    expect(matchesAny("Error: Failed to fetch canvas")).toBe(false);
    expect(matchesAny("ReferenceError: postEvent is not defined")).toBe(false);
  });
});
