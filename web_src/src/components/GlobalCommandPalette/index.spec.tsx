import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router-dom";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GlobalCommandPalette } from ".";
import { registerCanvasNodeSearchProvider } from "./canvasNodeSearchStore";

const {
  accountState,
  createCanvasMock,
  defaultAccount,
  defaultPermissions,
  featureState,
  navigateMock,
  permissionsState,
} = vi.hoisted(() => {
  const defaultAccount = {
    id: "account-1",
    name: "Ada Lovelace",
    email: "ada@example.com",
    avatar_url: "",
    installation_admin: true,
    has_password: true,
  };
  type Account = typeof defaultAccount;
  const defaultPermissions = [
    { resource: "canvases", action: "create" },
    { resource: "canvases", action: "read" },
    { resource: "canvases", action: "update" },
    { resource: "org", action: "read" },
    { resource: "members", action: "read" },
    { resource: "service_accounts", action: "read" },
    { resource: "groups", action: "read" },
    { resource: "roles", action: "read" },
    { resource: "integrations", action: "read" },
    { resource: "secrets", action: "read" },
  ];

  return {
    accountState: { account: defaultAccount as Account | null, loading: false },
    createCanvasMock: vi.fn(),
    defaultAccount,
    defaultPermissions,
    featureState: { managedAgentsEnabled: true },
    navigateMock: vi.fn(),
    permissionsState: { permissions: defaultPermissions },
  };
});

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof ReactRouterDom>("react-router-dom");
  return {
    ...actual,
    useNavigate: () => navigateMock,
  };
});

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: accountState.account,
    loading: accountState.loading,
    setupRequired: false,
  }),
}));

vi.mock("@/api-client", () => ({
  meMe: vi.fn(async () => ({
    data: {
      user: {
        permissions: permissionsState.permissions,
      },
    },
  })),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvases: () => ({
    data: [
      {
        metadata: {
          id: "canvas-1",
          name: "Deploy API",
          description: "Production deployment flow",
        },
      },
      {
        metadata: {
          id: "canvas-2",
          name: "Database Backups",
          description: "Nightly backup flow",
        },
      },
    ],
    isLoading: false,
  }),
  useCreateCanvas: () => ({
    mutateAsync: createCanvasMock,
    isPending: false,
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({
    data: {
      metadata: {
        id: "org-1",
        name: "Acme",
      },
    },
  }),
  useOrganizationUsage: () => ({
    data: { enabled: true },
    error: null,
  }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (feature: string) => feature === "claude_managed_agents" && featureState.managedAgentsEnabled,
    enabledExperimentalFeatures: featureState.managedAgentsEnabled ? ["claude_managed_agents"] : [],
  }),
}));

vi.mock("@/lib/canvasNameGenerator", () => ({
  generateCanvasName: () => "generated-canvas",
}));

let unregisterCanvasNodeSearchProvider: (() => void) | undefined;

function renderPalette(path = "/org-1/canvases/canvas-1") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <GlobalCommandPalette />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("GlobalCommandPalette", () => {
  beforeEach(() => {
    accountState.account = defaultAccount;
    accountState.loading = false;
    Element.prototype.scrollIntoView = vi.fn();
    createCanvasMock.mockReset();
    createCanvasMock.mockResolvedValue({ data: { canvas: { metadata: { id: "canvas-new" } } } });
    navigateMock.mockReset();
    permissionsState.permissions = [...defaultPermissions];
    featureState.managedAgentsEnabled = true;
  });

  afterEach(() => {
    unregisterCanvasNodeSearchProvider?.();
    unregisterCanvasNodeSearchProvider = undefined;
  });

  it("opens with the global shortcut and shows contextual commands", async () => {
    renderPalette();

    fireEvent.keyDown(document, { key: "k", metaKey: true });

    expect(await screen.findByPlaceholderText("Search components and commands")).toBeInTheDocument();
    expect(screen.getByText("New Canvas")).toBeInTheDocument();
    expect(screen.getByText("Console")).toBeInTheDocument();
    expect(screen.getByText("Agent")).toBeInTheDocument();
    expect(screen.getByText("Versions")).toBeInTheDocument();
    expect(screen.getByText("Organization Settings")).toBeInTheDocument();
    expect(screen.getByText("Installation Admin")).toBeInTheDocument();
  });

  it("hides the agent command when managed agents are disabled", async () => {
    featureState.managedAgentsEnabled = false;
    renderPalette();

    fireEvent.keyDown(document, { key: "k", metaKey: true });

    expect(await screen.findByPlaceholderText("Search components and commands")).toBeInTheDocument();
    expect(screen.queryByText("Agent")).not.toBeInTheDocument();
    expect(screen.getByText("Versions")).toBeInTheDocument();
  });

  it("opens current canvas tool tabs from commands", async () => {
    const user = userEvent.setup();
    const dispatchEventSpy = vi.spyOn(window, "dispatchEvent");
    renderPalette("/org-1/canvases/canvas-1?view=memory");

    fireEvent.keyDown(document, { key: "k", metaKey: true });
    await user.click(await screen.findByText("Versions"));

    await waitFor(() => {
      expect(navigateMock).toHaveBeenCalledWith("/org-1/canvases/canvas-1");
      expect(dispatchEventSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          type: "canvas-tool-sidebar:select-tab",
          detail: { tab: "versions" },
        }),
      );
    });

    dispatchEventSpy.mockRestore();
  });

  it("does not capture shortcuts before the account is available", () => {
    accountState.account = null;
    renderPalette("/login");

    const event = new KeyboardEvent("keydown", { key: "k", metaKey: true, cancelable: true });

    expect(document.dispatchEvent(event)).toBe(true);
    expect(screen.queryByPlaceholderText("What can we help with?")).not.toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search components and commands")).not.toBeInTheDocument();
  });

  it("opens organization settings as a nested command page", async () => {
    const user = userEvent.setup();
    renderPalette();

    fireEvent.keyDown(document, { key: "k", ctrlKey: true });
    await user.click(await screen.findByText("Organization Settings"));
    await user.click(await screen.findByText("Members"));

    expect(navigateMock).toHaveBeenCalledWith("/org-1/settings/members");
  });

  it("lists canvases when opening canvas settings outside a canvas route", async () => {
    const user = userEvent.setup();
    renderPalette("/org-1");

    fireEvent.keyDown(document, { key: "k", metaKey: true });
    await user.click(await screen.findByText("Canvas Settings"));
    await user.click(await screen.findByText("Database Backups"));

    expect(navigateMock).toHaveBeenCalledWith("/org-1/canvases/canvas-2/settings");
  });

  it("disables canvas settings commands without canvas update permission", async () => {
    permissionsState.permissions = defaultPermissions.filter(
      (permission) => !(permission.resource === "canvases" && permission.action === "update"),
    );
    renderPalette();

    fireEvent.keyDown(document, { key: "k", metaKey: true });

    await screen.findByPlaceholderText("Search components and commands");
    const settingsItems = screen.getAllByText("Canvas Settings").map((label) => label.closest("[cmdk-item]"));

    expect(settingsItems).toHaveLength(2);
    settingsItems.forEach((item) => expect(item).toHaveAttribute("data-disabled", "true"));
  });

  it("uses template route canvas ids for contextual canvas commands", async () => {
    renderPalette("/org-1/templates/canvas-1");

    fireEvent.keyDown(document, { key: "k", metaKey: true });

    expect(await screen.findByPlaceholderText("Search components and commands")).toBeInTheDocument();
    expect(screen.getByText("Console")).toBeInTheDocument();
  });

  it("creates a canvas with the quick shortcut", async () => {
    renderPalette();

    fireEvent.keyDown(document, { key: "k", metaKey: true });
    await waitFor(() => {
      expect(screen.getByText("New Canvas").closest("[cmdk-item]")).not.toHaveAttribute("data-disabled", "true");
    });
    fireEvent.keyDown(document, { key: "/", metaKey: true });

    await waitFor(() => {
      expect(createCanvasMock).toHaveBeenCalledWith({ name: "generated-canvas", method: "ui" });
    });
    expect(navigateMock).toHaveBeenCalledWith("/org-1/canvases/canvas-new");
  });

  it("searches canvas components from the root command palette", async () => {
    const user = userEvent.setup();
    const selectNode = vi.fn();
    unregisterCanvasNodeSearchProvider = registerCanvasNodeSearchProvider({
      searchNodes: (query) =>
        [
          {
            id: "node-api",
            label: "Deploy API",
            iconSlug: "box",
            keywords: ["deploy api", "node-api"],
          },
          {
            id: "node-worker",
            label: "Refresh Worker",
            iconSlug: "box",
            keywords: ["refresh worker", "node-worker"],
          },
        ].filter((node) => node.label.toLowerCase().includes(query.toLowerCase())),
      selectNode,
    });

    renderPalette();

    fireEvent.keyDown(document, { key: "k", metaKey: true });
    await user.type(await screen.findByPlaceholderText("Search components and commands"), "worker");
    await user.click(await screen.findByText("Refresh Worker"));

    expect(selectNode).toHaveBeenCalledWith("node-worker");
  });
});
