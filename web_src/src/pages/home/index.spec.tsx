import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { CanvasesCanvasSummary } from "@/api-client";
import type { ReactNode } from "react";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";

class MockResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

vi.stubGlobal("ResizeObserver", MockResizeObserver);

const { useCanvases, useDeleteCanvas, useCreateCanvas, useUpdateCanvasPreference } = vi.hoisted(() => ({
  useCanvases: vi.fn(),
  useDeleteCanvas: vi.fn(),
  useCreateCanvas: vi.fn(),
  useUpdateCanvasPreference: vi.fn(),
}));

const mutationMocks = vi.hoisted(() => ({
  deleteCanvas: vi.fn(),
  deleteCanvasAsync: vi.fn(),
  createCanvas: vi.fn(),
  createCanvasAsync: vi.fn(),
  updateCanvasPreference: vi.fn(),
}));

type CanAct = (resource: string, action: string) => boolean;

const permissionMocks = vi.hoisted(() => ({
  canAct: vi.fn<CanAct>(() => true),
}));

vi.mock("@/components/OrganizationMenuButton", () => ({
  OrganizationMenuButton: () => null,
}));

vi.mock("@/components/Dialog/dialog", () => ({
  Dialog: ({ children, open }: { children: ReactNode; open: boolean }) => (open ? <div>{children}</div> : null),
  DialogActions: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: ReactNode }) => <p>{children}</p>,
  DialogTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
}));

vi.mock("./EditAppModal", () => ({
  EditAppModal: () => null,
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "user-1", name: "Ada Lovelace" } }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: permissionMocks.canAct,
    isLoading: false,
  }),
}));

const experimentalFeatureMocks = vi.hoisted(() => ({
  has: vi.fn((_featureId?: string) => false),
  isLoading: false,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: experimentalFeatureMocks.has,
    enabledExperimentalFeatures: [],
    isLoading: experimentalFeatureMocks.isLoading,
  }),
}));

vi.mock("./useEditApp", () => ({
  useEditApp: () => ({
    editingCanvas: null,
    openEdit: vi.fn(),
    closeEdit: vi.fn(),
    saveApp: vi.fn(),
    isSaving: false,
    isOpen: false,
  }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  canvasKeys: {
    detail: (organizationId: string, canvasId: string) => ["canvases", "detail", organizationId, canvasId],
    list: (organizationId: string) => ["canvases", "list", organizationId],
  },
  useCanvases,
  useDeleteCanvas,
  useCreateCanvas,
  useUpdateCanvasPreference,
}));

import { HomePage } from "./index";
import { NewAppPage } from "./NewAppPage";

function makeCanvas(id: string, name: string, overrides: Partial<CanvasesCanvasSummary> = {}): CanvasesCanvasSummary {
  return {
    id,
    name,
    createdAt: "2026-05-05T00:00:00Z",
    createdBy: { name: "Ada Lovelace" },
    nodes: [],
    edges: [],
    ...overrides,
  } as CanvasesCanvasSummary;
}

function renderHome(initialEntries = ["/org-123"]) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={initialEntries}>
        <Routes>
          <Route path="/:organizationId">
            <Route index element={<HomePage />} />
            <Route path="apps/new" element={<NewAppPage />} />
            <Route path="apps/:canvasId" element={<div>Canvas editor</div>} />
            <Route path="workspaces" element={<div data-testid="workspaces-index">Workspaces</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("HomePage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
    vi.clearAllMocks();
    window.localStorage.clear();
    experimentalFeatureMocks.has.mockReturnValue(false);
    experimentalFeatureMocks.isLoading = false;
    permissionMocks.canAct.mockReturnValue(true);

    useDeleteCanvas.mockReturnValue({
      mutate: mutationMocks.deleteCanvas,
      mutateAsync: mutationMocks.deleteCanvasAsync,
      isPending: false,
    });
    useCreateCanvas.mockReturnValue({
      mutate: mutationMocks.createCanvas,
      mutateAsync: mutationMocks.createCanvasAsync,
      isPending: false,
    });
    useUpdateCanvasPreference.mockReturnValue({
      mutate: mutationMocks.updateCanvasPreference,
      isPending: false,
    });
  });

  it("uses the factory-first landing as the canvas creation entrypoint", async () => {
    const user = userEvent.setup();
    mutationMocks.createCanvasAsync.mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new" } } },
    });
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();

    await user.click(await screen.findByRole("button", { name: /create a blank app/i }));

    await waitFor(() => {
      expect(mutationMocks.createCanvasAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          name: expect.stringMatching(/^[a-z]+-[a-z]+$/),
          method: "ui",
        }),
      );
    });
  });

  it("does not redirect an empty home page to creation without create permission", () => {
    permissionMocks.canAct.mockImplementation((_resource: string, action: string) => action !== "create");
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();

    expect(screen.getByRole("heading", { name: "Apps" })).toBeInTheDocument();
    expect(screen.getByText("No apps yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create new app" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: /create a blank app/i })).not.toBeInTheDocument();
  });

  it("blocks direct navigation to the new app page without create permission", () => {
    permissionMocks.canAct.mockImplementation((_resource: string, action: string) => action !== "create");
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome(["/org-123/apps/new"]);

    expect(screen.getByTestId("permission-denied-page")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Permission denied" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /create a blank app/i })).not.toBeInTheDocument();
  });

  it("orders starred canvases first and requests star updates", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [
        makeCanvas("a-free", "A Free Canvas"),
        makeCanvas("starred", "Starred Canvas", {
          starred: true,
          starredAt: "2026-05-06T00:00:00Z",
        }),
      ],
      isLoading: false,
      error: null,
    });

    renderHome();

    expect(screen.queryByRole("heading", { name: "Pinned" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Pin app A Free Canvas")).not.toBeInTheDocument();
    expect(
      screen.getByText("Starred Canvas").compareDocumentPosition(screen.getByText("A Free Canvas")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    await user.click(screen.getByLabelText("Unstar app Starred Canvas"));
    expect(mutationMocks.updateCanvasPreference).toHaveBeenCalledWith({ canvasId: "starred", starred: false });
  });

  it("redirects org home to /workspaces when factories are on", () => {
    experimentalFeatureMocks.has.mockImplementation((featureId) => featureId === FEATURE_FACTORIES);
    renderHome(["/org-123"]);

    expect(screen.getByTestId("workspaces-index")).toBeInTheDocument();
  });

  it("redirects /apps/new to /workspaces when factories are on", () => {
    experimentalFeatureMocks.has.mockImplementation((featureId) => featureId === FEATURE_FACTORIES);
    renderHome(["/org-123/apps/new"]);

    expect(screen.getByTestId("workspaces-index")).toBeInTheDocument();
  });
});
