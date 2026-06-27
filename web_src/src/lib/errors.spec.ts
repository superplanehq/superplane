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
    it("matches Safari ReferenceErrors for short minified identifiers", () => {
      expect(looksLikeMinifiedReferenceError("Can't find variable: oU")).toBe(true);
      expect(looksLikeMinifiedReferenceError("Can't find variable: Gy")).toBe(true);
      expect(looksLikeMinifiedReferenceError("Can't find variable: a")).toBe(true);
      expect(looksLikeMinifiedReferenceError("ReferenceError: Can't find variable: oU")).toBe(true);
      expect(looksLikeMinifiedReferenceError("  Can't find variable: oU  ")).toBe(true);
    });

    it("matches V8 / Firefox ReferenceErrors for short minified identifiers", () => {
      expect(looksLikeMinifiedReferenceError("oU is not defined")).toBe(true);
      expect(looksLikeMinifiedReferenceError("ReferenceError: vl is not defined")).toBe(true);
      expect(looksLikeMinifiedReferenceError("$ is not defined")).toBe(true);
    });

    it("does not match descriptive identifier names that likely indicate real bugs", () => {
      expect(looksLikeMinifiedReferenceError("Can't find variable: getCanvasDashboard")).toBe(false);
      expect(looksLikeMinifiedReferenceError("Can't find variable: userId")).toBe(false);
      expect(looksLikeMinifiedReferenceError("loader is not defined")).toBe(false);
      expect(looksLikeMinifiedReferenceError("ReferenceError: window.something is not defined")).toBe(false);
    });

    it("does not match unrelated error messages", () => {
      expect(looksLikeMinifiedReferenceError("")).toBe(false);
      expect(looksLikeMinifiedReferenceError("TypeError: undefined is not an object")).toBe(false);
      expect(looksLikeMinifiedReferenceError("Failed to fetch")).toBe(false);
      expect(looksLikeMinifiedReferenceError("Can't find variable")).toBe(false);
    });
  });
});
