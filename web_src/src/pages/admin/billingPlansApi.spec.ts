import { afterEach, describe, expect, it, vi } from "bun:test";
import { billingCatalogRequest } from "./billingPlansApi";

afterEach(() => vi.unstubAllGlobals());

async function rejectionMessage(request: Promise<unknown>): Promise<string | null> {
  try {
    await request;
  } catch (error) {
    return error instanceof Error ? error.message : null;
  }
  return null;
}

describe("billingCatalogRequest", () => {
  it("reports the message from a structured billing error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => Response.json({ message: "Enter a different key." }, { status: 400 })),
    );

    expect(await rejectionMessage(billingCatalogRequest("billing-plans", "POST", {}))).toBe("Enter a different key.");
  });

  it("keeps the plain-text message from routes that have not moved to JSON", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("Polar seat price must match.", { status: 400 })),
    );

    expect(await rejectionMessage(billingCatalogRequest("organizations/org-1/seat-limit", "PUT", {}))).toBe(
      "Polar seat price must match.",
    );
  });
});
