import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type * as ApiClient from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { showErrorToast } from "@/lib/toast";
import { ReplayNodeModal } from "./ReplayNodeModal";

const replayNodeMock = vi.fn();

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReplayNode: (...args: unknown[]) => replayNodeMock(...args),
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea data-testid="monaco-stub" value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

function renderModal(onClose = vi.fn()) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ReplayNodeModal
          onClose={onClose}
          canvasId="canvas-1"
          nodeId="join"
          originalPayload={{ foo: "bar" }}
          sourceNodeId="lane-a"
        />
      </ThemeProvider>
    </QueryClientProvider>,
  );

  return { onClose };
}

describe("ReplayNodeModal API errors", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reports a refusal inline only, without also raising a toast for the same failure", async () => {
    const user = userEvent.setup();
    replayNodeMock.mockRejectedValue({
      code: 3,
      message: "replay refused: missing input(s) for distinct incoming source(s): lane-b",
      details: [],
    });
    const { onClose } = renderModal();

    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => {
      expect(screen.getByTestId("replay-api-error")).toHaveTextContent(
        "replay refused: missing input(s) for distinct incoming source(s): lane-b",
      );
    });
    expect(showErrorToast).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("falls back to the generic failure when the error carries no usable message", async () => {
    const user = userEvent.setup();
    replayNodeMock.mockRejectedValue(new Error("Failed to fetch"));
    renderModal();

    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => {
      expect(screen.getByTestId("replay-api-error")).toHaveTextContent("Failed to replay node");
    });
    expect(showErrorToast).not.toHaveBeenCalled();
  });

  it("clears the previous refusal once the payload is edited again", async () => {
    const user = userEvent.setup();
    replayNodeMock.mockRejectedValue({ code: 3, message: "replay refused: missing input(s)" });
    renderModal();

    await user.click(screen.getByTestId("launch-replay-button"));
    await waitFor(() => expect(screen.getByTestId("replay-api-error")).toBeInTheDocument());

    await user.clear(screen.getByTestId("monaco-stub"));
    await user.type(screen.getByTestId("monaco-stub"), '{{"foo":"baz"}');

    expect(screen.queryByTestId("replay-api-error")).not.toBeInTheDocument();
  });
});
