import { describe, expect, it } from "vitest";
import { getApiErrorMessage, getResponseErrorMessage } from "@/lib/errors";

describe("errors", () => {
  it("extracts nested api error messages", () => {
    expect(getApiErrorMessage({ response: { data: { message: "from response data" } } }, "fallback")).toBe(
      "from response data",
    );
    expect(getApiErrorMessage({ error: { message: "from nested error" } }, "fallback")).toBe("from nested error");
    expect(getApiErrorMessage(new Error("from error instance"), "fallback")).toBe("from error instance");
    expect(getApiErrorMessage({}, "fallback")).toBe("fallback");
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

  it("falls back when the response body is empty", async () => {
    const response = new Response("", { status: 500 });

    await expect(getResponseErrorMessage(response, "fallback")).resolves.toBe("fallback");
  });
});
