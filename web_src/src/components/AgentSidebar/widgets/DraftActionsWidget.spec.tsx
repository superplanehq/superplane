import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DraftActionsWidget } from "./DraftActionsWidget";

type FetchResponseInit = {
  ok: boolean;
  status: number;
  body: string;
};

function mockFetchResponse({ ok, status, body }: FetchResponseInit): ReturnType<typeof vi.fn> {
  const fetchMock = vi.fn().mockResolvedValue({
    ok,
    status,
    text: async () => body,
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("DraftActionsWidget", () => {
  const baseProps = {
    versionId: "ver-1",
    canvasId: "canvas-1",
    organizationId: "org-1",
    isEditing: false,
  };

  let errorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    errorSpy.mockRestore();
    vi.unstubAllGlobals();
  });

  it("dismisses silently when the API reports the draft is no longer publishable", async () => {
    mockFetchResponse({
      ok: false,
      status: 400,
      body: JSON.stringify({ code: 9, message: "only draft versions can be published", details: [] }),
    });
    const onDismiss = vi.fn();

    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(onDismiss).toHaveBeenCalledTimes(1));

    // Critically: we don't log to console.error for an expected 4xx outcome, so
    // captureConsoleIntegration doesn't ship a Sentry event for this case.
    expect(errorSpy).not.toHaveBeenCalled();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("shows an inline error (and stays mounted) on other 4xx responses without logging", async () => {
    mockFetchResponse({
      ok: false,
      status: 400,
      body: JSON.stringify({ code: 9, message: "change management is enabled for this canvas", details: [] }),
    });
    const onDismiss = vi.fn();

    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("change management is enabled for this canvas");
    expect(onDismiss).not.toHaveBeenCalled();
    expect(errorSpy).not.toHaveBeenCalled();
  });

  it("logs a console.error for unexpected 5xx responses so they reach Sentry", async () => {
    mockFetchResponse({
      ok: false,
      status: 500,
      body: JSON.stringify({ message: "internal error" }),
    });
    const onDismiss = vi.fn();

    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(errorSpy).toHaveBeenCalledTimes(1));
    expect(errorSpy.mock.calls[0]?.[0]).toContain("publish failed:");
    expect(onDismiss).not.toHaveBeenCalled();
    expect(await screen.findByRole("alert")).toHaveTextContent("internal error");
  });

  it("invokes onDismiss after a successful publish", async () => {
    const fetchMock = mockFetchResponse({ ok: true, status: 200, body: "" });
    const onDismiss = vi.fn();

    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(onDismiss).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/canvases/canvas-1/versions/ver-1/publish",
      expect.objectContaining({ method: "PATCH" }),
    );
    expect(errorSpy).not.toHaveBeenCalled();
  });
});
