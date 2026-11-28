import { describe, expect, it } from "bun:test";
import { FIREWALL_BLOCKED_MESSAGE, getApiErrorMessage, getApiErrorReason, getResponseErrorMessage } from "@/lib/errors";

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

  it("uses the fallback when the API hides the failure as internal error", () => {
    expect(
      getApiErrorMessage(
        { response: { data: { message: "internal error" } } },
        "SuperPlane could not create the Sentry intake.",
      ),
    ).toBe("SuperPlane could not create the Sentry intake.");
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

  it("explains a firewall block when a 403 response is an HTML page", async () => {
    const firewallPage =
      '<!doctype html><meta charset="utf-8"><meta name=viewport content="width=device-width, initial-scale=1"><title>403</title>403 Forbidden';

    expect(getApiErrorMessage(Object.assign(new Error(firewallPage), { status: 403 }), "fallback")).toBe(
      FIREWALL_BLOCKED_MESSAGE,
    );
    await expect(
      getResponseErrorMessage(
        new Response(firewallPage, { status: 403, headers: { "Content-Type": "text/html" } }),
        "fallback",
      ),
    ).resolves.toBe(FIREWALL_BLOCKED_MESSAGE);
  });

  it("keeps the API message for a JSON permission error", () => {
    expect(getApiErrorMessage(Object.assign(new Error("permission denied"), { status: 403 }), "fallback")).toBe(
      "permission denied",
    );
  });

  it("falls back when the response body is empty", async () => {
    const response = new Response("", { status: 500 });

    await expect(getResponseErrorMessage(response, "fallback")).resolves.toBe("fallback");
  });

  it("extracts the SuperPlane error reason", () => {
    const info = {
      "@type": "type.googleapis.com/google.rpc.ErrorInfo",
      domain: "superplane.com",
      reason: "SOME_REASON",
    };

    expect(getApiErrorReason(Object.assign(new Error("failed"), { details: [{ foo: "bar" }, info] }))).toBe(
      "SOME_REASON",
    );
    expect(getApiErrorReason({ details: [{ ...info, domain: "example.com" }] })).toBeNull();
    expect(getApiErrorReason({ details: "SOME_REASON" })).toBeNull();
    expect(getApiErrorReason(new Error("failed"))).toBeNull();
    expect(getApiErrorReason(null)).toBeNull();
  });
});
