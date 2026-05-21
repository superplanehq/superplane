import { describe, expect, it } from "vitest";
import { getApiErrorMessage, getResponseErrorMessage, isTransientHttpError, summarizeError } from "@/lib/errors";

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
});

describe("isTransientHttpError", () => {
  it("treats HTML response bodies as transient", () => {
    const htmlError = `<!DOCTYPE html>
<html lang="en-US">
  <head><title>superplane.com | 502: Bad gateway</title></head>
  <body>Bad gateway</body>
</html>`;

    expect(isTransientHttpError(htmlError)).toBe(true);
    expect(isTransientHttpError("<html><head></head><body>hi</body></html>")).toBe(true);
  });

  it("treats browser network failures as transient", () => {
    expect(isTransientHttpError(new TypeError("Failed to fetch"))).toBe(true);
    expect(isTransientHttpError(new TypeError("Load failed"))).toBe(true);
    expect(isTransientHttpError({ name: "AbortError" })).toBe(true);
  });

  it("treats transient HTTP statuses as transient", () => {
    for (const status of [0, 401, 403, 408, 429, 502, 503, 504]) {
      expect(isTransientHttpError({ status })).toBe(true);
      expect(isTransientHttpError({ response: { status } })).toBe(true);
    }
  });

  it("does not treat real application errors as transient", () => {
    expect(isTransientHttpError({ status: 400 })).toBe(false);
    expect(isTransientHttpError({ response: { status: 422 } })).toBe(false);
    expect(isTransientHttpError(new Error("validation failed"))).toBe(false);
    expect(isTransientHttpError("conflict resolving canvas changes")).toBe(false);
    expect(isTransientHttpError(undefined)).toBe(false);
    expect(isTransientHttpError(null)).toBe(false);
  });
});

describe("summarizeError", () => {
  it("collapses HTML response bodies to a short label", () => {
    const htmlError = `<!DOCTYPE html>
<html lang="en-US">
  <head><title>superplane.com | 502: Bad gateway</title></head>
  <body>Bad gateway</body>
</html>`;

    expect(summarizeError(htmlError)).toBe("HTML response body");
  });

  it("includes the HTTP status when available", () => {
    expect(summarizeError({ status: 503 })).toBe("HTTP 503");
    expect(summarizeError({ response: { status: 504 } })).toBe("HTTP 504");
  });

  it("returns the error message for Error instances", () => {
    expect(summarizeError(new Error("validation failed"))).toBe("validation failed");
  });

  it("truncates very long messages", () => {
    const longString = "x".repeat(500);
    const summary = summarizeError(longString, 50);
    expect(summary.length).toBeLessThanOrEqual(50);
    expect(summary.endsWith("…")).toBe(true);
  });

  it("handles non-string, non-error values", () => {
    expect(summarizeError(undefined)).toBe("Unknown error");
    expect(summarizeError(null)).toBe("Unknown error");
    expect(summarizeError({ foo: "bar" })).toBe('{"foo":"bar"}');
  });
});
