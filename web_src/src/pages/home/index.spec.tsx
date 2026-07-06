import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasFoldersCanvasFolder, CanvasesCanvasSummary } from "@/api-client";
import type { ReactNode } from "react";
import { showErrorToast } from "@/lib/toast";

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
  useCanvasFolders,
  useDeleteCanvas,
  useCreateCanvas,
  useCreateCanvasFolder,
  useUpdateCanvasFolder,
  useMoveCanvasFolder,
  useDeleteCanvasFolder,
  useUpdateCanvasFolderMembership,
  useUpdateCanvasPreference,
} = vi.hoisted(() => ({
  useCanvases: vi.fn(),
  useCanvasFolders: vi.fn(),
  useDeleteCanvas: vi.fn(),
  useCreateCanvas: vi.fn(),
  useCreateCanvasFolder: vi.fn(),
  useUpdateCanvasFolder: vi.fn(),
  useMoveCanvasFolder: vi.fn(),
  useDeleteCanvasFolder: vi.fn(),
  useUpdateCanvasFolderMembership: vi.fn(),
  useUpdateCanvasPreference: vi.fn(),
}));

const mutationMocks = vi.hoisted(() => ({
  deleteCanvas: vi.fn(),
  deleteCanvasAsync: vi.fn(),
  createCanvas: vi.fn(),
  createCanvasAsync: vi.fn(),
  createCanvasFolder: vi.fn(),
  updateCanvasFolder: vi.fn(),
  moveCanvasFolder: vi.fn(),
  deleteCanvasFolder: vi.fn(),
  updateCanvasFolderMembership: vi.fn(),
  updateCanvasPreference: vi.fn(),
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
    canAct: () => true,
    isLoading: false,
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
  CANVAS_FOLDER_COLORS: ["blue", "green", "purple", "slate", "orange"],
  DEFAULT_CANVAS_FOLDER_COLOR: "blue",
  canvasKeys: {
    detail: (organizationId: string, canvasId: string) => ["canvases", "detail", organizationId, canvasId],
    list: (organizationId: string) => ["canvases", "list", organizationId],
  },
  useCanvases,
  useCanvasFolders,
  useDeleteCanvas,
  useCreateCanvas,
  useCreateCanvasFolder,
  useUpdateCanvasFolder,
  useMoveCanvasFolder,
  useDeleteCanvasFolder,
  useUpdateCanvasFolderMembership,
  useUpdateCanvasPreference,
}));

import { HomePage } from "./index";
import { NewAppPage } from "./NewAppPage";

function makeCanvas(
  id: string,
  name: string,
  canvasFolderId?: string,
  overrides: Partial<CanvasesCanvasSummary> = {},
): CanvasesCanvasSummary {
  return {
    id,
    name,
    folderId: canvasFolderId,
    createdAt: "2026-05-05T00:00:00Z",
    createdBy: { name: "Ada Lovelace" },
    nodes: [],
    edges: [],
    ...overrides,
  } as CanvasesCanvasSummary;
}

function makeFolder(
  id: string,
  title: string,
  backgroundColor = "blue",
  canvasIds: string[] = [],
): CanvasFoldersCanvasFolder {
  return {
    metadata: { id },
    spec: { title, backgroundColor, canvases: canvasIds.map((canvasId) => ({ id: canvasId })) },
  } as CanvasFoldersCanvasFolder;
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
          <Route path="/:organizationId">
            <Route index element={<HomePage />} />
            <Route path="apps/new" element={<NewAppPage />} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("HomePage canvas folders", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    mutationMocks.createCanvasFolder.mockResolvedValue({ data: { folder: { metadata: { id: "new-folder" } } } });
    mutationMocks.updateCanvasFolder.mockResolvedValue({});
    mutationMocks.moveCanvasFolder.mockResolvedValue({});
    mutationMocks.deleteCanvasFolder.mockResolvedValue({});
    mutationMocks.updateCanvasFolderMembership.mockResolvedValue({});

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
    useCreateCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.createCanvasFolder, isPending: false });
    useUpdateCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.updateCanvasFolder, isPending: false });
    useMoveCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.moveCanvasFolder, isPending: false });
    useDeleteCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.deleteCanvasFolder, isPending: false });
    useUpdateCanvasFolderMembership.mockReturnValue({
      mutateAsync: mutationMocks.updateCanvasFolderMembership,
      isPending: false,
    });
    useUpdateCanvasPreference.mockReturnValue({
      mutate: mutationMocks.updateCanvasPreference,
      isPending: false,
    });
  });

  it("uses the zero-state as the canvas creation entrypoint", async () => {
    const user = userEvent.setup();
    mutationMocks.createCanvasAsync.mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new" } } },
    });
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();

    await user.click(screen.getByRole("button", { name: /start from scratch/i }));

    await waitFor(() => {
      expect(mutationMocks.createCanvasAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          name: expect.stringMatching(/^[a-z]+-[a-z]+$/),
          method: "ui",
        }),
      );
    });
  });

  it("renders folders before free canvases using the manual folder order", () => {
    useCanvases.mockReturnValue({
      data: [
        makeCanvas("z-free", "Z Free Canvas"),
        makeCanvas("a-free", "A Free Canvas"),
        makeCanvas("foldered", "Foldered Canvas", "folder-2"),
      ],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-2", "Zulu", "green"), makeFolder("folder-1", "Alpha", "blue")],
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

  it("keeps folders with unloaded member canvases visible", () => {
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green", ["missing-canvas"])],
      isLoading: false,
      error: null,
    });

    renderHome();

    const deploymentsSection = screen.getByText("Deployments").closest("section")!;
    expect(deploymentsSection).toBeInTheDocument();
    expect(within(deploymentsSection).getByLabelText("Folder actions")).toBeInTheDocument();
  });

  it("orders preferred canvases and requests preference updates", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [
        makeCanvas("z-free", "Z Free Canvas", undefined, {
          folderId: "folder-1",
          pinned: true,
          pinnedAt: "2026-05-06T00:00:00Z",
        }),
        makeCanvas("starred", "Starred Canvas", undefined, {
          starred: true,
          starredAt: "2026-05-06T00:00:00Z",
        }),
        makeCanvas("a-free", "A Free Canvas"),
      ],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "blue", ["z-free"])],
      isLoading: false,
      error: null,
    });

    renderHome();

    const pinnedSection = screen.getByRole("heading", { name: "Pinned" }).closest("section")!;
    const deploymentsSection = screen.getByText("Deployments").closest("section")!;
    expect(within(pinnedSection).getByText("Z Free Canvas")).toBeInTheDocument();
    expect(within(deploymentsSection).getByText("Z Free Canvas")).toBeInTheDocument();
    expect(
      within(pinnedSection).getByText("Z Free Canvas").compareDocumentPosition(screen.getByText("A Free Canvas")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      screen.getByText("Starred Canvas").compareDocumentPosition(screen.getByText("A Free Canvas")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    await user.click(within(pinnedSection).getByLabelText("Unpin app Z Free Canvas"));
    await user.click(screen.getByLabelText("Unstar app Starred Canvas"));
    expect(mutationMocks.updateCanvasPreference).toHaveBeenCalledWith({ canvasId: "z-free", pinned: false });
    expect(mutationMocks.updateCanvasPreference).toHaveBeenCalledWith({ canvasId: "starred", starred: false });
  });

  it("moves a folder up from the folder menu", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Alpha"), makeFolder("folder-2", "Beta")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const betaSection = screen.getByText("Beta").closest("section")!;
    await user.click(within(betaSection).getByLabelText("Folder actions"));
    await user.click(await screen.findByText("Move Up"));

    await waitFor(() => {
      expect(mutationMocks.moveCanvasFolder).toHaveBeenCalledWith({
        folderId: "folder-2",
        direction: "DIRECTION_UP",
      });
    });
  });

  it("updates folder color from the folder menu", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green")],
      isLoading: false,
      error: null,
    });

    renderHome();
    await user.click(screen.getByLabelText("Folder actions"));
    await user.hover(screen.getByText("Background"));
    fireEvent.click(await screen.findByLabelText("violet folder color"));

    await waitFor(() => {
      expect(mutationMocks.updateCanvasFolder).toHaveBeenCalledWith({
        folderId: "folder-1",
        title: "Deployments",
        backgroundColor: "purple",
      });
    });
  });

  it("renames a folder inline", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green")],
      isLoading: false,
      error: null,
    });

    renderHome();

    await user.click(screen.getByRole("button", { name: "Rename folder Deployments" }));
    const input = screen.getByLabelText("Folder name");
    await user.clear(input);
    await user.type(input, "Operations{enter}");

    await waitFor(() => {
      expect(mutationMocks.updateCanvasFolder).toHaveBeenCalledWith({
        folderId: "folder-1",
        title: "Operations",
        backgroundColor: "green",
      });
    });
  });

  it("shows folder rename action in the folder menu", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green")],
      isLoading: false,
      error: null,
    });

    renderHome();

    await user.click(screen.getByLabelText("Folder actions"));
    expect(await screen.findByText("Change folder name")).toBeInTheDocument();
  });

  it("adds a canvas to an existing folder", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("canvas-1", "Free Canvas")],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const card = screen.getByLabelText("Open canvas Free Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));
    await user.hover(screen.getByText("Add to Folder"));
    fireEvent.click(await screen.findByRole("menuitem", { name: /deployments/i }));

    await waitFor(() => {
      expect(mutationMocks.updateCanvasFolderMembership).toHaveBeenCalledWith({
        folderId: "folder-1",
        title: "Deployments",
        backgroundColor: "blue",
        canvasIds: ["canvas-1"],
      });
    });
  });

  it("uses move copy when a canvas is already in a folder", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("foldered", "Foldered Canvas", "folder-1")],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments"), makeFolder("folder-2", "Operations")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const card = screen.getByLabelText("Open canvas Foldered Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));

    expect(await screen.findByText("Move to Folder")).toBeInTheDocument();
    expect(screen.queryByText("Add to Folder")).not.toBeInTheDocument();
  });

  it("creates a folder and assigns the current canvas to it", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("canvas-1", "Free Canvas")],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();
    const card = screen.getByLabelText("Open canvas Free Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));
    await user.hover(screen.getByText("Add to Folder"));
    const input = await screen.findByPlaceholderText("New folder name");
    fireEvent.change(input, { target: { value: "Release" } });
    fireEvent.submit(input.closest("form")!);

    await waitFor(() => {
      expect(mutationMocks.createCanvasFolder).toHaveBeenCalledWith({
        title: "Release",
        backgroundColor: "blue",
      });
      expect(mutationMocks.updateCanvasFolderMembership).toHaveBeenCalledWith({
        folderId: "new-folder",
        title: "Release",
        backgroundColor: "blue",
        canvasIds: ["canvas-1"],
      });
    });
  });

  it("shows assignment error when folder creation succeeds but adding the canvas fails", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("canvas-1", "Free Canvas")],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({ data: [], isLoading: false, error: null });
    mutationMocks.updateCanvasFolderMembership.mockRejectedValue(new Error("Failed to fetch"));

    renderHome();
    const card = screen.getByLabelText("Open canvas Free Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));
    await user.hover(screen.getByText("Add to Folder"));
    const input = await screen.findByPlaceholderText("New folder name");
    fireEvent.change(input, { target: { value: "Release" } });
    fireEvent.submit(input.closest("form")!);

    await waitFor(() => {
      expect(mutationMocks.createCanvasFolder).toHaveBeenCalledWith({
        title: "Release",
        backgroundColor: "blue",
      });
      expect(showErrorToast).toHaveBeenCalledWith("Folder created, but failed to add canvas to it");
    });
  });

  it("does not create a folder with a duplicate name", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("canvas-1", "Free Canvas")],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "blue", ["foldered"])],
      isLoading: false,
      error: null,
    });

    renderHome();
    const card = screen.getByLabelText("Open canvas Free Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));
    await user.hover(screen.getByText("Add to Folder"));
    const input = await screen.findByPlaceholderText("New folder name");
    fireEvent.change(input, { target: { value: " deployments " } });

    expect(await screen.findByText("Folder name already exists")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create Folder" })).toBeDisabled();

    fireEvent.submit(input.closest("form")!);

    expect(mutationMocks.createCanvasFolder).not.toHaveBeenCalled();
    expect(mutationMocks.updateCanvasFolderMembership).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalledWith("Folder name already exists");
  });

  it("removes a canvas from its folder", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [makeCanvas("foldered", "Foldered Canvas", "folder-1")],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const card = screen.getByLabelText("Open canvas Foldered Canvas").parentElement!;
    await user.click(within(card).getByLabelText("Canvas actions"));

    expect(await screen.findByText("Move to Folder")).toBeInTheDocument();
    expect(await screen.findByText("Remove from Folder")).toBeInTheDocument();
    await user.click(screen.getByText("Remove from Folder"));
    expect(mutationMocks.updateCanvasFolderMembership).toHaveBeenCalledWith({
      folderId: "folder-1",
      title: "Deployments",
      backgroundColor: "blue",
      canvasIds: [],
    });
  });
});
