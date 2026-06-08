import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const showErrorToast = vi.fn();
const showSuccessToast = vi.fn();
const sentryWithScope = vi.fn();
const sentryCaptureException = vi.fn();

vi.mock("@/lib/toast", () => ({
  showErrorToast: (...args: unknown[]) => showErrorToast(...args),
  showSuccessToast: (...args: unknown[]) => showSuccessToast(...args),
}));

vi.mock("@/sentry", () => ({
  Sentry: {
    withScope: (
      callback: (scope: { setTag: ReturnType<typeof vi.fn>; setExtra: ReturnType<typeof vi.fn> }) => void,
    ) => {
      const scope = { setTag: vi.fn(), setExtra: vi.fn() };
      sentryWithScope(scope);
      callback(scope);
    },
    captureException: (...args: unknown[]) => sentryCaptureException(...args),
  },
}));

import { DraftActionsWidget } from "./DraftActionsWidget";

const baseProps = {
  versionId: "version-1",
  canvasId: "canvas-1",
  organizationId: "org-1",
  isEditing: false,
};

function mockFetch(handler: (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>) {
  globalThis.fetch = vi.fn(handler) as unknown as typeof fetch;
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("DraftActionsWidget", () => {
  beforeEach(() => {
    showErrorToast.mockClear();
    showSuccessToast.mockClear();
    sentryWithScope.mockClear();
    sentryCaptureException.mockClear();
    vi.spyOn(console, "error").mockImplementation(() => undefined);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("dismisses and shows a success toast on a successful publish", async () => {
    mockFetch(async () => new Response(null, { status: 200 }));
    const onDismiss = vi.fn();
    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(onDismiss).toHaveBeenCalled());
    expect(showSuccessToast).toHaveBeenCalledWith("Draft published.");
    expect(sentryCaptureException).not.toHaveBeenCalled();
  });

  it("silently dismisses when a 4xx indicates the draft is stale", async () => {
    mockFetch(async () => jsonResponse(412, { code: 9, message: "only draft versions can be published" }));
    const onDismiss = vi.fn();
    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(onDismiss).toHaveBeenCalled());
    expect(showErrorToast).not.toHaveBeenCalled();
    expect(sentryCaptureException).not.toHaveBeenCalled();
  });

  it("shows a toast but does not report Sentry on other 4xx errors", async () => {
    mockFetch(async () => jsonResponse(400, { code: 3, message: "canvas name conflict" }));
    const onDismiss = vi.fn();
    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(showErrorToast).toHaveBeenCalledWith("canvas name conflict"));
    expect(onDismiss).not.toHaveBeenCalled();
    expect(sentryCaptureException).not.toHaveBeenCalled();
  });

  it("captures structured Sentry context on 5xx publish failures", async () => {
    mockFetch(async () => jsonResponse(500, { code: 13, message: "internal error", details: [] }));
    const onDismiss = vi.fn();
    render(<DraftActionsWidget {...baseProps} onDismiss={onDismiss} />);

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(sentryCaptureException).toHaveBeenCalledTimes(1));
    expect(showErrorToast).toHaveBeenCalledWith("internal error");
    expect(onDismiss).not.toHaveBeenCalled();

    const scope = sentryWithScope.mock.calls[0][0] as {
      setTag: ReturnType<typeof vi.fn>;
      setExtra: ReturnType<typeof vi.fn>;
    };
    expect(scope.setTag).toHaveBeenCalledWith("draft.action", "publish");
    expect(scope.setTag).toHaveBeenCalledWith("draft.status", "500");
    expect(scope.setExtra).toHaveBeenCalledWith("canvasId", "canvas-1");
    expect(scope.setExtra).toHaveBeenCalledWith("versionId", "version-1");
    expect(scope.setExtra).toHaveBeenCalledWith("apiMessage", "internal error");
  });

  it("captures network failures to Sentry as the underlying Error", async () => {
    const cause = new Error("Failed to fetch");
    mockFetch(async () => {
      throw cause;
    });
    render(<DraftActionsWidget {...baseProps} />);

    fireEvent.click(screen.getByRole("button", { name: /discard/i }));

    await waitFor(() => expect(sentryCaptureException).toHaveBeenCalledTimes(1));
    expect(sentryCaptureException).toHaveBeenCalledWith(cause);
    expect(showErrorToast).toHaveBeenCalledWith("Failed to discard draft.");
  });
});
