import { describe, expect, it } from "vitest";
import { getApiErrorMessage, getResponseErrorMessage, looksLikeMinifiedReferenceError } from "@/lib/errors";

describe("errors", () => {
  it("extracts nested api error messages", () => {
    expect(getApiErrorMessage({ response: { data: { message: "from response data" } } }, "fallback")).toBe(
      "from response data",
    );
    expect(getApiErrorMessage({ error: { message: "from nested error" } }, "fallback")).toBe("from nested error");
    expect(getApiErrorMessage(new Error("from error instance"), "fallback")).toBe("from error instance");
    expect(getApiErrorMessage({}, "fallback")).toBe("fallback");
  });

  it("falls back when the api error message is an HTML error page", () => {
    const htmlError = `<!DOCTYPE html>
<html lang="en-US">
  <head><title>superplane.com | 502: Bad gateway</title></head>
  <body>Bad gateway</body>
</html>`;

    expect(getApiErrorMessage(htmlError, "Failed to save changes to the canvas")).toBe(
      "Failed to save changes to the canvas",
    );
    expect(
      getApiErrorMessage({ response: { data: { message: htmlError } } }, "Failed to save changes to the canvas"),
    ).toBe("Failed to save changes to the canvas");
  });

  it("falls back when the api error message is a generic browser network failure", () => {
    expect(getApiErrorMessage(new Error("Failed to fetch"), "Failed to save changes to the canvas")).toBe(
      "Failed to save changes to the canvas",
    );
    expect(
      getApiErrorMessage(
        { response: { data: { message: "NetworkError when attempting to fetch resource." } } },
        "Failed to emit event",
      ),
    ).toBe("Failed to emit event");
  });

  it("extracts a message from a JSON error response", async () => {
    const response = new Response(JSON.stringify({ message: "account organization limit exceeded" }), {
      status: 429,
      headers: {
        "Content-Type": "application/json",
      },
    });

    await expect(getResponseErrorMessage(response, "fallback")).resolves.toBe("account organization limit exceeded");
  });

  it("returns the plain text body when the response is not JSON", async () => {
    const response = new Response("account organization limit exceeded\n", {
      status: 429,
      headers: {
        "Content-Type": "text/plain",
      },
    });

    await expect(getResponseErrorMessage(response, "fallback")).resolves.toBe("account organization limit exceeded");
  });

  it("falls back when the response body is an HTML error page", async () => {
    const response = new Response(
      `<!DOCTYPE html>
<html lang="en-US">
  <head><title>superplane.com | 502: Bad gateway</title></head>
  <body>Bad gateway</body>
</html>`,
      {
        status: 502,
        headers: {
          "Content-Type": "text/html",
        },
      },
    );

    await expect(getResponseErrorMessage(response, "fallback")).resolves.toBe("fallback");
  });

  it("falls back when the response body is empty", async () => {
    const response = new Response("", { status: 500 });

    await expect(getResponseErrorMessage(response, "fallback")).resolves.toBe("fallback");
  });

  describe("looksLikeMinifiedReferenceError", () => {
    it("matches Safari-style ReferenceError messages for minified identifiers", () => {
      expect(looksLikeMinifiedReferenceError("Can't find variable: Gy")).toBe(true);
      expect(looksLikeMinifiedReferenceError("ReferenceError: Can't find variable: vl")).toBe(true);
      expect(looksLikeMinifiedReferenceError("  Can't find variable: a  ")).toBe(true);
      expect(looksLikeMinifiedReferenceError("Can't find variable: $1")).toBe(true);
    });

    it("matches V8 / Firefox style ReferenceError messages for minified identifiers", () => {
      expect(looksLikeMinifiedReferenceError("Gy is not defined")).toBe(true);
      expect(looksLikeMinifiedReferenceError("ReferenceError: vl is not defined")).toBe(true);
      expect(looksLikeMinifiedReferenceError("_a is not defined")).toBe(true);
    });

    it("ignores ReferenceError messages that look like real source identifiers", () => {
      // 4+ char identifiers indicate either a real bug in our source or a
      // global like `gtag` / `posthog` that we want to know about.
      expect(looksLikeMinifiedReferenceError("Can't find variable: posthog")).toBe(false);
      expect(looksLikeMinifiedReferenceError("posthog is not defined")).toBe(false);
      expect(looksLikeMinifiedReferenceError("Can't find variable: ResizeObserver")).toBe(false);
    });

    it("ignores unrelated error messages", () => {
      expect(looksLikeMinifiedReferenceError("Failed to fetch")).toBe(false);
      expect(looksLikeMinifiedReferenceError("Cannot read properties of undefined (reading 'foo')")).toBe(false);
      expect(looksLikeMinifiedReferenceError("")).toBe(false);
    });
  });
});
