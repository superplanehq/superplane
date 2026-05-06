import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvas, CanvasesCanvasGroup } from "@/api-client";
import type { ReactNode } from "react";

vi.stubGlobal(
  "ResizeObserver",
  class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
);

const {
  useCanvases,
  useCanvasGroups,
  useDeleteCanvas,
  useCreateCanvasGroup,
  useUpdateCanvasGroup,
  useUpdateCanvasGroupPosition,
  useDeleteCanvasGroup,
  useUpdateCanvasGroupMembership,
} = vi.hoisted(() => ({
  useCanvases: vi.fn(),
  useCanvasGroups: vi.fn(),
  useDeleteCanvas: vi.fn(),
  useCreateCanvasGroup: vi.fn(),
  useUpdateCanvasGroup: vi.fn(),
  useUpdateCanvasGroupPosition: vi.fn(),
  useDeleteCanvasGroup: vi.fn(),
  useUpdateCanvasGroupMembership: vi.fn(),
}));

const mutationMocks = vi.hoisted(() => ({
  deleteCanvas: vi.fn(),
  deleteCanvasAsync: vi.fn(),
  createCanvasGroup: vi.fn(),
  updateCanvasGroup: vi.fn(),
  updateCanvasGroupPosition: vi.fn(),
  deleteCanvasGroup: vi.fn(),
  updateCanvasGroupMembership: vi.fn(),
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

vi.mock("@/components/CreateCanvasModal", () => ({
  CreateCanvasModal: () => null,
}));

vi.mock("@/contexts/AccountContext", () => ({
  useAccount: () => ({ account: { id: "user-1", name: "Ada Lovelace" } }),
}));

vi.mock("@/contexts/PermissionsContext", () => ({
  usePermissions: () => ({
    canAct: () => true,
    isLoading: false,
  }),
}));

vi.mock("./useCreateCanvasModalState", () => ({
  useCreateCanvasModalState: () => ({
    onOpenEdit: vi.fn(),
  }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  CANVAS_GROUP_COLORS: ["color_1", "color_2", "color_3", "color_4", "color_5", "color_6"],
  DEFAULT_CANVAS_GROUP_COLOR: "color_1",
  canvasKeys: {
    detail: (organizationId: string, canvasId: string) => ["canvases", "detail", organizationId, canvasId],
  },
  useCanvases,
  useCanvasGroups,
  useDeleteCanvas,
  useCreateCanvasGroup,
  useUpdateCanvasGroup,
  useUpdateCanvasGroupPosition,
  useDeleteCanvasGroup,
  useUpdateCanvasGroupMembership,
}));

import HomePage from "./index";

function makeCanvas(id: string, name: string, canvasGroupId?: string): CanvasesCanvas {
  return {
    metadata: {
      id,
      name,
      canvasGroupId,
      createdAt: "2026-05-05T00:00:00Z",
    },
    spec: { nodes: [], edges: [] },
  } as CanvasesCanvas;
}

function makeGroup(id: string, title: string, backgroundColor = "color_1"): CanvasesCanvasGroup {
  return {
    metadata: { id },
    spec: { title, backgroundColor },
  } as CanvasesCanvasGroup;
}

function renderHome() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/org-123"]}>
        <Routes>
          <Route path="/:organizationId" element={<HomePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("HomePage canvas groups", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mutationMocks.createCanvasGroup.mockResolvedValue({ data: { group: { metadata: { id: "new-group" } } } });
    mutationMocks.updateCanvasGroup.mockResolvedValue({});
    mutationMocks.updateCanvasGroupPosition.mockResolvedValue({});
    mutationMocks.deleteCanvasGroup.mockResolvedValue({});
    mutationMocks.updateCanvasGroupMembership.mockResolvedValue({});

    useDeleteCanvas.mockReturnValue({
      mutate: mutationMocks.deleteCanvas,
      mutateAsync: mutationMocks.deleteCanvasAsync,
      isPending: false,
    });
    useCreateCanvasGroup.mockReturnValue({ mutateAsync: mutationMocks.createCanvasGroup, isPending: false });
    useUpdateCanvasGroup.mockReturnValue({ mutateAsync: mutationMocks.updateCanvasGroup, isPending: false });
    useUpdateCanvasGroupPosition.mockReturnValue({
      mutateAsync: mutationMocks.updateCanvasGroupPosition,
      isPending: false,
    });
    useDeleteCanvasGroup.mockReturnValue({ mutateAsync: mutationMocks.deleteCanvasGroup, isPending: false });
    useUpdateCanvasGroupMembership.mockReturnValue({
      mutateAsync: mutationMocks.updateCanvasGroupMembership,
      isPending: false,
    });
  });

  it("uses the toolbar as the canvas creation entrypoint", () => {
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasGroups.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();

    expect(screen.getByRole("link", { name: /new canvas/i })).toHaveAttribute("href", "/org-123/canvases/new");
    expect(screen.queryByText("Point & Click")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Grid view")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("List view")).not.toBeInTheDocument();
  });

  it("renders groups before free canvases using the manual group order", () => {
    useCanvases.mockReturnValue({
      data: [
        makeCanvas("z-free", "Z Free Canvas"),
        makeCanvas("a-free", "A Free Canvas"),
        makeCanvas("grouped", "Grouped Canvas", "group-2"),
      ],
      isLoading: false,
      error: null,
    });
    useCanvasGroups.mockReturnValue({
      data: [makeGroup("group-2", "Zulu", "color_2"), makeGroup("group-1", "Alpha", "color_1")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const zulu = screen.getByText("Zulu");
    const alpha = screen.getByText("Alpha");
    const aFreeCanvas = screen.getByText("A Free Canvas");
    const zFreeCanvas = screen.getByText("Z Free Canvas");

    expect(zulu.compareDocumentPosition(alpha) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(zulu.compareDocumentPosition(aFreeCanvas) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(aFreeCanvas.compareDocumentPosition(zFreeCanvas) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("moves a group up from the group menu", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasGroups.mockReturnValue({
      data: [makeGroup("group-1", "Alpha"), makeGroup("group-2", "Beta")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const betaSection = screen.getByText("Beta").closest("section")!;
    await user.click(within(betaSection).getByLabelText("Group actions"));
    await user.click(await screen.findByText("Move Up"));

    await waitFor(() => {
      expect(mutationMocks.updateCanvasGroupPosition).toHaveBeenCalledWith({
        groupId: "group-2",
        direction: "DIRECTION_UP",
      });
    });
  });

  it("updates group color from the group menu", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasGroups.mockReturnValue({
      data: [makeGroup("group-1", "Deployments", "color_2")],
      isLoading: false,
      error: null,
    });

    renderHome();
    await user.click(screen.getByLabelText("Group actions"));
    await user.hover(screen.getByText("Background"));
    fireEvent.click(await screen.findByLabelText("violet group color"));

    await waitFor(() => {
      expect(mutationMocks.updateCanvasGroup).toHaveBeenCalledWith({
        groupId: "group-1",
        title: "Deployments",
        backgroundColor: "color_3",
      });
    });
  });

  it("renames a group inline", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasGroups.mockReturnValue({
      data: [makeGroup("group-1", "Deployments", "color_2")],
      isLoading: false,
      error: null,
    });

    renderHome();

    await user.click(screen.getByRole("button", { name: "Rename group Deployments" }));
    const input = screen.getByLabelText("Group name");
    await user.clear(input);
    await user.type(input, "Operations{enter}");

    await waitFor(() => {
      expect(mutationMocks.updateCanvasGroup).toHaveBeenCalledWith({
        groupId: "group-1",
        title: "Operations",
        backgroundColor: "color_2",
      });
    });
  });

  it("shows group rename action in the group menu", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasGroups.mockReturnValue({
      data: [makeGroup("group-1", "Deployments", "color_2")],
      isLoading: false,
      error: null,
    });

    renderHome();

    await user.click(screen.getByLabelText("Group actions"));
    expect(await screen.findByText("Change group name")).toBeInTheDocument();
  });

  it("adds a canvas to an existing group", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("canvas-1", "Free Canvas")],
      isLoading: false,
      error: null,
    });
    useCanvasGroups.mockReturnValue({
      data: [makeGroup("group-1", "Deployments")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const card = screen.getByLabelText("Open canvas Free Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));
    await user.hover(screen.getByText("Add to Group"));
    fireEvent.click(await screen.findByRole("menuitem", { name: /deployments/i }));

    await waitFor(() => {
      expect(mutationMocks.updateCanvasGroupMembership).toHaveBeenCalledWith({
        canvasId: "canvas-1",
        groupId: "group-1",
      });
    });
  });

  it("creates a group and assigns the current canvas to it", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("canvas-1", "Free Canvas")],
      isLoading: false,
      error: null,
    });
    useCanvasGroups.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();
    const card = screen.getByLabelText("Open canvas Free Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));
    await user.hover(screen.getByText("Add to Group"));
    const input = await screen.findByPlaceholderText("New group name");
    fireEvent.change(input, { target: { value: "Release" } });
    fireEvent.submit(input.closest("form")!);

    await waitFor(() => {
      expect(mutationMocks.createCanvasGroup).toHaveBeenCalledWith({
        title: "Release",
        backgroundColor: "color_1",
      });
      expect(mutationMocks.updateCanvasGroupMembership).toHaveBeenCalledWith({
        canvasId: "canvas-1",
        groupId: "new-group",
      });
    });
  });

  it("removes a canvas from its group", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("grouped", "Grouped Canvas", "group-1")],
      isLoading: false,
      error: null,
    });
    useCanvasGroups.mockReturnValue({
      data: [makeGroup("group-1", "Deployments")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const card = screen.getByLabelText("Open canvas Grouped Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));

    expect(await screen.findByText("Remove from Group")).toBeInTheDocument();
    await user.click(screen.getByText("Remove from Group"));
    expect(mutationMocks.updateCanvasGroupMembership).toHaveBeenCalledWith({ canvasId: "grouped" });
  });
});
