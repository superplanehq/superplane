import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { canvasesPublishCanvasVersion, canvasesDeleteCanvasVersion, showErrorToast } = vi.hoisted(() => ({
  canvasesPublishCanvasVersion: vi.fn(),
  canvasesDeleteCanvasVersion: vi.fn(),
  showErrorToast: vi.fn(),
}));

vi.mock("../../../api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasesPublishCanvasVersion,
    canvasesDeleteCanvasVersion,
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast,
  showSuccessToast: vi.fn(),
  showWarningToast: vi.fn(),
}));

import { DraftActionsWidget } from "./DraftActionsWidget";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

describe("DraftActionsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls onDismiss after a successful publish", async () => {
    const queryClient = createQueryClient();
    const onDismiss = vi.fn();
    canvasesPublishCanvasVersion.mockResolvedValue({
      data: { version: { metadata: { id: "v-1" } } },
    });

    render(
      <DraftActionsWidget
        versionId="v-1"
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        onDismiss={onDismiss}
      />,
      { wrapper: createWrapper(queryClient) },
    );

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(onDismiss).toHaveBeenCalledTimes(1));
    expect(canvasesPublishCanvasVersion).toHaveBeenCalledOnce();
    expect(showErrorToast).not.toHaveBeenCalled();
  });

  it("shows an error toast and does not log to console.error when publish fails", async () => {
    const queryClient = createQueryClient();
    const onDismiss = vi.fn();
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    canvasesPublishCanvasVersion.mockRejectedValue({
      response: { data: { code: 13, message: "internal error", details: [] } },
    });

    render(
      <DraftActionsWidget
        versionId="v-1"
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        onDismiss={onDismiss}
      />,
      { wrapper: createWrapper(queryClient) },
    );

    fireEvent.click(screen.getByRole("button", { name: /publish/i }));

    await waitFor(() => expect(showErrorToast).toHaveBeenCalledTimes(1));
    expect(showErrorToast).toHaveBeenCalledWith("internal error");
    expect(onDismiss).not.toHaveBeenCalled();
    expect(consoleErrorSpy).not.toHaveBeenCalled();

    consoleErrorSpy.mockRestore();
  });

  it("calls onDismiss after a successful discard", async () => {
    const queryClient = createQueryClient();
    const onDismiss = vi.fn();
    canvasesDeleteCanvasVersion.mockResolvedValue({ data: {} });

    render(
      <DraftActionsWidget
        versionId="v-1"
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        onDismiss={onDismiss}
      />,
      { wrapper: createWrapper(queryClient) },
    );

    fireEvent.click(screen.getByRole("button", { name: /discard/i }));

    await waitFor(() => expect(onDismiss).toHaveBeenCalledTimes(1));
    expect(canvasesDeleteCanvasVersion).toHaveBeenCalledOnce();
    expect(showErrorToast).not.toHaveBeenCalled();
  });

  it("shows an error toast when discard fails without leaking to console", async () => {
    const queryClient = createQueryClient();
    const onDismiss = vi.fn();
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    canvasesDeleteCanvasVersion.mockRejectedValue(new Error("network down"));

    render(
      <DraftActionsWidget
        versionId="v-1"
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        onDismiss={onDismiss}
      />,
      { wrapper: createWrapper(queryClient) },
    );

    fireEvent.click(screen.getByRole("button", { name: /discard/i }));

    await waitFor(() => expect(showErrorToast).toHaveBeenCalledTimes(1));
    expect(showErrorToast).toHaveBeenCalledWith("network down");
    expect(onDismiss).not.toHaveBeenCalled();
    expect(consoleErrorSpy).not.toHaveBeenCalled();

    consoleErrorSpy.mockRestore();
  });
});
