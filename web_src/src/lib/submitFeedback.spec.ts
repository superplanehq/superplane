import { describe, expect, it, vi } from "vitest";
import { FeedbackRequestError, isAllowedFeedbackAttachment, submitFeedback } from "./submitFeedback";

describe("isAllowedFeedbackAttachment", () => {
  it("accepts a PNG under the size limit", () => {
    const file = new File(["png"], "shot.png", { type: "image/png" });
    expect(isAllowedFeedbackAttachment(file)).toBe(true);
  });

  it("rejects a disallowed type", () => {
    const file = new File(["exe"], "payload.exe", { type: "application/octet-stream" });
    expect(isAllowedFeedbackAttachment(file)).toBe(false);
  });
});

describe("submitFeedback", () => {
  it("posts multipart form data", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) });
    vi.stubGlobal("fetch", fetchMock);

    const file = new File(["png"], "shot.png", { type: "image/png" });
    await submitFeedback({
      organizationId: "org-1",
      category: "bug",
      details: "The canvas did not load.",
      pagePath: "/org-1/apps/deploy",
      file,
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/me/feedback");
    expect(init.method).toBe("POST");
    expect((init.headers as Record<string, string>)["x-organization-id"]).toBe("org-1");
    const body = init.body as FormData;
    expect(body.get("category")).toBe("bug");
    expect(body.get("details")).toBe("The canvas did not load.");
    expect(body.get("page_path")).toBe("/org-1/apps/deploy");
    expect(body.get("file")).toBe(file);
  });

  it("throws the server message when the request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        json: async () => ({ message: "Select a feedback category." }),
      }),
    );

    await expect(submitFeedback({ organizationId: "org-1", category: "bug", details: "x" })).rejects.toBeInstanceOf(
      FeedbackRequestError,
    );
  });
});
