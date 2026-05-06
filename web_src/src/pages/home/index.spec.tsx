import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasFoldersCanvasFolder, CanvasesCanvas } from "@/api-client";
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
  useCreateCanvasFolder,
  useUpdateCanvasFolder,
  useMoveCanvasFolder,
  useDeleteCanvasFolder,
  useUpdateCanvasFolderMembership,
} = vi.hoisted(() => ({
  useCanvases: vi.fn(),
  useCanvasFolders: vi.fn(),
  useDeleteCanvas: vi.fn(),
  useCreateCanvasFolder: vi.fn(),
  useUpdateCanvasFolder: vi.fn(),
  useMoveCanvasFolder: vi.fn(),
  useDeleteCanvasFolder: vi.fn(),
  useUpdateCanvasFolderMembership: vi.fn(),
}));

const mutationMocks = vi.hoisted(() => ({
  deleteCanvas: vi.fn(),
  deleteCanvasAsync: vi.fn(),
  createCanvasFolder: vi.fn(),
  updateCanvasFolder: vi.fn(),
  moveCanvasFolder: vi.fn(),
  deleteCanvasFolder: vi.fn(),
  updateCanvasFolderMembership: vi.fn(),
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
  CANVAS_FOLDER_COLORS: ["color_1", "color_2", "color_3", "color_4", "color_5", "color_6"],
  DEFAULT_CANVAS_FOLDER_COLOR: "color_1",
  canvasKeys: {
    detail: (organizationId: string, canvasId: string) => ["canvases", "detail", organizationId, canvasId],
  },
  useCanvases,
  useCanvasFolders,
  useDeleteCanvas,
  useCreateCanvasFolder,
  useUpdateCanvasFolder,
  useMoveCanvasFolder,
  useDeleteCanvasFolder,
  useUpdateCanvasFolderMembership,
}));

import HomePage from "./index";

function makeCanvas(id: string, name: string, canvasFolderId?: string): CanvasesCanvas {
  return {
    metadata: {
      id,
      name,
      folderId: canvasFolderId,
      createdAt: "2026-05-05T00:00:00Z",
    },
    spec: { nodes: [], edges: [] },
  } as CanvasesCanvas;
}

function makeFolder(
  id: string,
  title: string,
  backgroundColor = "color_1",
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
          <Route path="/:organizationId" element={<HomePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("HomePage canvas folders", () => {
  beforeEach(() => {
    vi.clearAllMocks();
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
    useCreateCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.createCanvasFolder, isPending: false });
    useUpdateCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.updateCanvasFolder, isPending: false });
    useMoveCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.moveCanvasFolder, isPending: false });
    useDeleteCanvasFolder.mockReturnValue({ mutateAsync: mutationMocks.deleteCanvasFolder, isPending: false });
    useUpdateCanvasFolderMembership.mockReturnValue({
      mutateAsync: mutationMocks.updateCanvasFolderMembership,
      isPending: false,
    });
  });

  it("uses the toolbar as the canvas creation entrypoint", () => {
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();

    expect(screen.getByRole("link", { name: /new canvas/i })).toHaveAttribute("href", "/org-123/canvases/new");
    expect(screen.queryByText("Point & Click")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Grid view")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("List view")).not.toBeInTheDocument();
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
      data: [makeFolder("folder-2", "Zulu", "color_2"), makeFolder("folder-1", "Alpha", "color_1")],
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
      data: [makeFolder("folder-1", "Deployments", "color_2")],
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
        backgroundColor: "color_3",
      });
    });
  });

  it("renames a folder inline", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "color_2")],
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
        backgroundColor: "color_2",
      });
    });
  });

  it("shows folder rename action in the folder menu", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "color_2")],
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
        backgroundColor: "color_1",
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
        backgroundColor: "color_1",
      });
      expect(mutationMocks.updateCanvasFolderMembership).toHaveBeenCalledWith({
        folderId: "new-folder",
        title: "Release",
        backgroundColor: "color_1",
        canvasIds: ["canvas-1"],
      });
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
      data: [makeFolder("folder-1", "Deployments", "color_1", ["foldered"])],
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
      backgroundColor: "color_1",
      canvasIds: [],
    });
  });
});
