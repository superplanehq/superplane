import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import type * as SdkGen from "@/api-client/sdk.gen";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { init, identify, capture, reset } = vi.hoisted(() => ({
  init: vi.fn(),
  identify: vi.fn(),
  capture: vi.fn(),
  reset: vi.fn(),
}));

const { canvasesCreateCanvas } = vi.hoisted(() => ({
  canvasesCreateCanvas: vi.fn(),
}));

vi.mock("posthog-js", () => ({
  default: { init, identify, capture, reset },
}));

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: ReactNode; to: string }) => <a href={to}>{children}</a>,
  useNavigate: () => vi.fn(),
  useParams: () => ({ organizationId: "org-123" }),
}));

vi.mock("@/api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal<typeof SdkGen>();
  return { ...actual, canvasesCreateCanvas };
});

vi.mock("@/components/OrganizationMenuButton", () => ({
  OrganizationMenuButton: () => null,
}));

vi.mock("@/components/CanvasCreation/CLIPanel", () => ({
  CLIPanel: () => null,
}));

vi.mock("@/components/CanvasCreation/AgentPanel", () => ({
  AgentPanel: () => null,
}));

vi.mock("./ImportYamlDialog", () => ({
  ImportYamlDialog: () => null,
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

import { CreateCanvasPage } from "./CreateCanvasPage";

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <CreateCanvasPage />
    </QueryClientProvider>,
  );
}

describe("CreateCanvasPage analytics", () => {
  beforeEach(() => {
    capture.mockClear();
  });

  it("captures canvas created on successful submission", async () => {
    const user = userEvent.setup();
    canvasesCreateCanvas.mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-123" } } },
    });

    renderPage();

    await user.type(screen.getByTestId("canvas-name-input"), "My Canvas");
    await user.click(screen.getByTestId("create-canvas-button"));

    await waitFor(() => {
      expect(capture).toHaveBeenCalledWith("canvas:canvas_create", {
        canvas_id: "canvas-123",
        organization_id: "org-123",
      });
    });
  });

  it("does not capture when canvas creation fails", async () => {
    const user = userEvent.setup();
    canvasesCreateCanvas.mockRejectedValue(new Error("Server error"));

    renderPage();

    await user.type(screen.getByTestId("canvas-name-input"), "My Canvas");
    await user.click(screen.getByTestId("create-canvas-button"));

    await waitFor(() => expect(screen.getByText(/Unable to create canvas/i)).toBeInTheDocument());
    expect(capture).not.toHaveBeenCalled();
  });
});
