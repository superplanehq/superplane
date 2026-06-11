import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

import { DraftActionsWidget } from "./DraftActionsWidget";

const showErrorToast = vi.fn();
const showSuccessToast = vi.fn();

vi.mock("@/lib/toast", () => ({
  showErrorToast: (message: string) => showErrorToast(message),
  showSuccessToast: (message: string) => showSuccessToast(message),
}));

const baseProps = {
  versionId: "ver-1",
  canvasId: "canvas-1",
  organizationId: "org-1",
  isEditing: false,
};

describe("DraftActionsWidget", () => {
  const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

  beforeEach(() => {
    showErrorToast.mockReset();
    showSuccessToast.mockReset();
    consoleErrorSpy.mockClear();
    consoleWarnSpy.mockClear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a user-friendly toast when the publish endpoint returns an HTML 502 page and does not log the raw HTML body", async () => {
    const htmlBody = `<!DOCTYPE html>\n<html lang="en-US"><head><title>superplane.com | 502: Bad gateway</title></head><body>Bad gateway</body></html>`;

    const fetchMock = vi.fn().mockResolvedValue(
      new Response(htmlBody, {
        status: 502,
        headers: { "Content-Type": "text/html" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    render(<DraftActionsWidget {...baseProps} />);

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledTimes(1);
    });

    expect(showErrorToast).toHaveBeenCalledWith("Failed to publish draft.");
    expect(consoleErrorSpy).not.toHaveBeenCalled();
    expect(consoleWarnSpy).not.toHaveBeenCalled();
  });

  it("uses the API error message when the publish endpoint returns a JSON error", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ message: "draft is no longer valid" }), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    render(<DraftActionsWidget {...baseProps} />);

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith("draft is no longer valid");
    });

    expect(consoleErrorSpy).not.toHaveBeenCalled();
    expect(consoleWarnSpy).not.toHaveBeenCalled();
  });

  it("falls back to a user-friendly toast when discard fails with a network error", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));
    vi.stubGlobal("fetch", fetchMock);

    render(<DraftActionsWidget {...baseProps} />);

    fireEvent.click(screen.getByRole("button", { name: /discard/i }));

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith("Failed to discard draft.");
    });

    expect(consoleErrorSpy).not.toHaveBeenCalled();
    expect(consoleWarnSpy).not.toHaveBeenCalled();
  });

  it("invokes onDismiss when publish succeeds", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const onDismiss = vi.fn();

    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => {
      expect(onDismiss).toHaveBeenCalledTimes(1);
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/canvases/canvas-1/versions/ver-1/publish",
      expect.objectContaining({ method: "PATCH" }),
    );
    expect(showErrorToast).not.toHaveBeenCalled();
  });
});
