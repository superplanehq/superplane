/* eslint-disable max-lines */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasFoldersCanvasFolder, CanvasesCanvasSummary } from "@/api-client";
import type { ReactNode } from "react";
import { showErrorToast } from "@/lib/toast";

class MockResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

vi.stubGlobal("ResizeObserver", MockResizeObserver);

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
  normalizeCanvasFolderColor: (value?: string) => {
    if (value === "yellow") {
      return "slate";
    }

    return ["blue", "green", "purple", "slate", "orange"].includes(value || "") ? value : "blue";
  },
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
import { InstallProgressPanel } from "./InstallProgressPanel";
import { NewAppPage } from "./NewAppPage";
import type { AppEntry } from "./AppDetailModal";

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
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function renderInstallProgressPanel(app: AppEntry) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/org-123/apps/new"]}>
        <InstallProgressPanel
          app={app}
          organizationId="org-123"
          skipPreviewFetch
          preloadedIntegrations={[]}
          preloadedParams={[]}
          onClose={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("HomePage canvas folders", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
    vi.clearAllMocks();
    window.localStorage.clear();
    permissionMocks.canAct.mockReturnValue(true);
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

  it("does not redirect an empty home page to creation without create permission", () => {
    permissionMocks.canAct.mockImplementation((_resource: string, action: string) => action !== "create");
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome();

    expect(screen.getByRole("heading", { name: "Apps" })).toBeInTheDocument();
    expect(screen.getByText("No apps yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create new app" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: /start from scratch/i })).not.toBeInTheDocument();
  });

  it("blocks direct navigation to the new app page without create permission", () => {
    permissionMocks.canAct.mockImplementation((_resource: string, action: string) => action !== "create");
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({ data: [], isLoading: false, error: null });

    renderHome(["/org-123/apps/new"]);

    expect(screen.getByRole("heading", { name: "404" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /start from scratch/i })).not.toBeInTheDocument();
  });

  it("does not install an app without create permission if the install action is invoked", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    permissionMocks.canAct.mockImplementation((_resource: string, action: string) => action !== "create");

    renderInstallProgressPanel({
      repo: "github.com/superplanehq/example-app",
      icon: "",
      title: "Example App",
      description: "",
      integrations: [],
      tags: [],
      requirements: [],
      agentInstructions: "",
    });

    await user.click(screen.getByRole("button", { name: "Just take me there" }));

    expect(fetchMock).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalledWith("You don't have permission to create canvases.");
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

  it("orders starred canvases first and requests star updates", async () => {
    const user = userEvent.setup();
    useCanvases.mockReturnValue({
      data: [
        makeCanvas("a-free", "A Free Canvas"),
        makeCanvas("starred", "Starred Canvas", undefined, {
          starred: true,
          starredAt: "2026-05-06T00:00:00Z",
        }),
      ],
      isLoading: false,
      error: null,
    });
    useCanvasFolders.mockReturnValue({ data: [], isLoading: false, error: null });

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

  it("opens the new app page scoped to a folder", async () => {
    const user = userEvent.setup();
    mutationMocks.createCanvasAsync.mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new" } } },
    });
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green", ["existing-canvas"])],
      isLoading: false,
      error: null,
    });

    renderHome();
    const deploymentsSection = screen.getByText("Deployments").closest("section")!;
    await user.click(within(deploymentsSection).getByLabelText("Create app in folder Deployments"));
    expect(await screen.findByRole("heading", { name: "Create New App in Deployments Folder" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /start from scratch/i }));

    await waitFor(() => {
      expect(mutationMocks.createCanvasAsync).toHaveBeenCalledWith({
        name: expect.stringMatching(/^[a-z]+-[a-z]+$/),
        method: "ui",
      });
      expect(mutationMocks.updateCanvasFolderMembership).toHaveBeenCalledWith({
        folderId: "folder-1",
        title: "Deployments",
        backgroundColor: "green",
        canvasIds: ["existing-canvas", "canvas-new"],
      });
    });
  });

  it("opens a folder-scoped new app when folder membership update fails", async () => {
    const user = userEvent.setup();
    mutationMocks.createCanvasAsync.mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new" } } },
    });
    mutationMocks.updateCanvasFolderMembership.mockRejectedValue(new Error("Failed to fetch"));
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const deploymentsSection = screen.getByText("Deployments").closest("section")!;
    await user.click(within(deploymentsSection).getByLabelText("Create app in folder Deployments"));
    await user.click(await screen.findByRole("button", { name: /start from scratch/i }));

    expect(await screen.findByText("Canvas editor")).toBeInTheDocument();
    expect(showErrorToast).toHaveBeenCalledWith("App created, but failed to add it to folder");
  });

  it("opens a folder-scoped installed app when folder membership update fails", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn();
    fetchMock
      .mockResolvedValueOnce(new Response(JSON.stringify({ integrations: [], installParams: [] }), { status: 200 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ canvasId: "canvas-installed", organizationId: "org-123" }), { status: 200 }),
      );
    vi.stubGlobal("fetch", fetchMock);
    mutationMocks.updateCanvasFolderMembership.mockRejectedValue(new Error("Failed to fetch"));
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green")],
      isLoading: false,
      error: null,
    });

    renderHome(["/org-123/apps/new?folderId=folder-1"]);
    await user.click((await screen.findAllByRole("button", { name: "Install" }))[0]);
    await user.click(await screen.findByRole("button", { name: "Just take me there" }));

    expect(await screen.findByText("Canvas editor")).toBeInTheDocument();
    expect(showErrorToast).toHaveBeenCalledWith("App installed, but failed to add it to folder");
  });

  it("disables folder header creation without update permission", async () => {
    permissionMocks.canAct.mockImplementation((_resource: string, action: string) => action !== "update");
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green")],
      isLoading: false,
      error: null,
    });

    renderHome();
    const deploymentsSection = screen.getByText("Deployments").closest("section")!;

    expect(within(deploymentsSection).getByLabelText("Create app in folder Deployments")).toBeDisabled();
  });

  it("does not create from a folder-scoped new app URL without update permission", async () => {
    const user = userEvent.setup();
    permissionMocks.canAct.mockImplementation((_resource: string, action: string) => action !== "update");
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [makeFolder("folder-1", "Deployments", "green")],
      isLoading: false,
      error: null,
    });

    renderHome(["/org-123/apps/new?folderId=folder-1"]);
    await user.click(await screen.findByRole("button", { name: /start from scratch/i }));

    expect(mutationMocks.createCanvasAsync).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalledWith("You don't have permission to update canvases.");
  });

  it("does not create outside a folder while folder context is loading", async () => {
    useCanvases.mockReturnValue({ data: [], isLoading: false, error: null });
    useCanvasFolders.mockReturnValue({
      data: [],
      isLoading: true,
      error: null,
    });

    renderHome(["/org-123/apps/new?folderId=folder-1"]);

    expect(await screen.findByRole("button", { name: /start from scratch/i })).toBeDisabled();
    expect(screen.getAllByRole("button", { name: "Install" })[0]).toBeDisabled();
    expect(mutationMocks.createCanvasAsync).not.toHaveBeenCalled();
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
