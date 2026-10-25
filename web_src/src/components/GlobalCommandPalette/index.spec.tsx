import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { unmockedPackage } from "@/test/unmockedModule";

import { GlobalCommandPalette } from ".";
import { registerCanvasNodeSearchProvider } from "./canvasNodeSearchStore";
import { openGlobalCommandPalette } from "./controller";

const {
  accountState,
  createCanvasMock,
  defaultAccount,
  defaultPermissions,
  navigateMock,
  paletteQueryState,
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
    { resource: "members", action: "create" },
    { resource: "api_keys", action: "read" },
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
    navigateMock: vi.fn(),
    paletteQueryState: {
      apiKeysCalls: 0,
      canvasEnabledValues: [] as Array<boolean | undefined>,
      integrationsCalls: 0,
      loading: false,
      organizationEnabledValues: [] as Array<boolean | undefined>,
    },
    permissionsState: { permissions: defaultPermissions },
  };
});

vi.mock("react-router", () => {
  const actual = unmockedPackage<typeof ReactRouterDom>("react-router/dist/development/index.js");
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
    refreshAccount: async () => undefined,
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
  useCanvases: (_organizationId: string, options?: { enabled?: boolean }) => {
    paletteQueryState.canvasEnabledValues.push(options?.enabled);
    return {
      data: [
        {
          id: "canvas-1",
          name: "Deploy API",
          description: "Production deployment flow",
        },
        {
          id: "canvas-2",
          name: "Database Backups",
          description: "Nightly backup flow",
        },
      ],
      isLoading: paletteQueryState.loading,
    };
  },
  useCreateCanvas: () => ({
    mutateAsync: createCanvasMock,
    isPending: false,
  }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => false,
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: (_organizationId: string, enabled?: boolean) => {
    paletteQueryState.organizationEnabledValues.push(enabled);
    return {
      data: {
        metadata: {
          id: "org-1",
          name: "Acme",
        },
      },
    };
  },
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => {
    paletteQueryState.integrationsCalls += 1;
    return {
      data: [
        {
          metadata: { id: "int-1", name: "puppies-github", integrationName: "github" },
          status: { state: "ready" },
        },
        {
          metadata: { id: "int-2", name: "deploy-alerts", integrationName: "slack" },
          status: { state: "ready" },
        },
      ],
      isLoading: paletteQueryState.loading,
    };
  },
}));

vi.mock("@/hooks/useApiKeys", () => ({
  useAPIKeys: () => {
    paletteQueryState.apiKeysCalls += 1;
    return {
      data: [{ id: "api-key-1", name: "deploy-bot" }],
      isLoading: false,
    };
  },
}));

vi.mock("@/lib/canvasNameGenerator", () => ({
  generateCanvasName: () => "generated-canvas",
}));

function openPalette() {
  openGlobalCommandPalette();
}

function declaredPointerEvents(element: Element) {
  let current: Element | null = element;
  while (current) {
    const pointerEvents = window.getComputedStyle(current).pointerEvents;
    if (pointerEvents && pointerEvents !== "inherit" && pointerEvents !== "unset") {
      return pointerEvents;
    }
    current = current.parentElement;
  }
  return "auto";
}

async function waitUntilPointerEventsAllow(element: Element) {
  await waitFor(() => {
    expect(declaredPointerEvents(element)).not.toBe("none");
  });
}

function setupUser() {
  const user = userEvent.setup();
  return {
    async click(element: Element) {
      if (!element.closest("[data-disabled='true']")) {
        await waitUntilPointerEventsAllow(element);
      }
      await user.click(element);
    },
    async type(element: Element, text: string) {
      await waitUntilPointerEventsAllow(element);
      await user.type(element, text);
    },
  };
}

function renderPalette(path = "/org-1") {
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

let unregisterCanvasNodeSearchProvider: (() => void) | null = null;

describe("GlobalCommandPalette", () => {
  beforeEach(() => {
    accountState.account = defaultAccount;
    accountState.loading = false;
    Element.prototype.scrollIntoView = vi.fn();
    createCanvasMock.mockReset();
    createCanvasMock.mockResolvedValue({ data: { canvas: { metadata: { id: "canvas-new" } } } });
    navigateMock.mockReset();
    paletteQueryState.apiKeysCalls = 0;
    paletteQueryState.canvasEnabledValues = [];
    paletteQueryState.integrationsCalls = 0;
    paletteQueryState.loading = false;
    paletteQueryState.organizationEnabledValues = [];
    permissionsState.permissions = [...defaultPermissions];
  });

  afterEach(() => {
    unregisterCanvasNodeSearchProvider?.();
    unregisterCanvasNodeSearchProvider = null;
  });

  it("opens and shows quick links", async () => {
    renderPalette();

    openPalette();

    expect(await screen.findByPlaceholderText("Find apps, integrations, and commands...")).toBeInTheDocument();
    expect(screen.getByText("New App")).toBeInTheDocument();
    expect(await screen.findByText("Invite by email")).toBeInTheDocument();
    expect(screen.getByText("Apps")).toBeInTheDocument();
    expect(screen.getByText("Integrations")).toBeInTheDocument();
    expect(screen.getByText("Go to Docs")).toBeInTheDocument();
    expect(screen.getByText("Sign Out")).toBeInTheDocument();
  });

  it("keeps palette-only queries disabled until the palette opens", async () => {
    renderPalette();

    expect(paletteQueryState.apiKeysCalls).toBe(0);
    expect(paletteQueryState.integrationsCalls).toBe(0);
    expect(paletteQueryState.canvasEnabledValues).not.toContain(true);
    expect(paletteQueryState.organizationEnabledValues).not.toContain(true);

    openPalette();

    expect(await screen.findByPlaceholderText("Find apps, integrations, and commands...")).toBeInTheDocument();
    expect(paletteQueryState.apiKeysCalls).toBeGreaterThan(0);
    expect(paletteQueryState.integrationsCalls).toBeGreaterThan(0);
    expect(paletteQueryState.canvasEnabledValues).toContain(true);
    expect(paletteQueryState.organizationEnabledValues).toContain(true);
  });

  it("shows a loading state while searchable data loads", async () => {
    const user = setupUser();
    paletteQueryState.loading = true;
    renderPalette();

    openPalette();
    const input = await screen.findByPlaceholderText("Find apps, integrations, and commands...");
    await user.type(input, "missing");

    expect(await screen.findByText("Loading results…")).toBeInTheDocument();

    paletteQueryState.loading = false;
    await user.type(input, " result");

    expect(await screen.findByText("No results found.")).toBeInTheDocument();
  });

  it("does not open or close with Cmd+K or Ctrl+K", async () => {
    renderPalette();

    fireEvent.keyDown(document, { key: "k", metaKey: true });
    fireEvent.keyDown(document, { key: "k", ctrlKey: true });

    expect(screen.queryByPlaceholderText("Find apps, integrations, and commands...")).not.toBeInTheDocument();

    openPalette();
    const input = await screen.findByPlaceholderText("Find apps, integrations, and commands...");
    input.focus();
    fireEvent.keyDown(input, { key: "k", metaKey: true });
    fireEvent.keyDown(document, { key: "k", ctrlKey: true });

    expect(screen.getByPlaceholderText("Find apps, integrations, and commands...")).toBeInTheDocument();
  });

  it("does not open before the account is available", () => {
    accountState.account = null;
    renderPalette("/login");

    openPalette();

    expect(screen.queryByPlaceholderText("Find apps, integrations, and commands...")).not.toBeInTheDocument();
  });

  it("expands app list when clicking Apps", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.click(await screen.findByText("Apps"));

    expect(await screen.findByText("Deploy API")).toBeInTheDocument();
    expect(screen.getByText("Database Backups")).toBeInTheDocument();
  });

  it("navigates to an app when selected from expanded list", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.click(await screen.findByText("Apps"));
    await user.click(await screen.findByText("Database Backups"));

    expect(navigateMock).toHaveBeenCalledWith("/org-1/apps/canvas-2");
  });

  it("searches apps by name", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "Deploy");

    expect(await screen.findByText("Deploy API")).toBeInTheDocument();
  });

  it("searches apps by description", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "Production");

    expect(await screen.findByText("Deploy API")).toBeInTheDocument();
  });

  it("searches integrations by name", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "puppies");

    expect(await screen.findByText("puppies-github")).toBeInTheDocument();
  });

  it("searches integrations by provider name", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "slack");

    expect(await screen.findByText("deploy-alerts")).toBeInTheDocument();
  });

  it("searches API keys by name", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "deploy-bot");

    expect(await screen.findByText("deploy-bot")).toBeInTheDocument();
  });

  it("does not match every result by shared category labels", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "service");

    expect(screen.queryByText("deploy-bot")).not.toBeInTheDocument();
    expect(await screen.findByText("No results found.")).toBeInTheDocument();
  });

  it("searches canvas nodes from the canvas page", async () => {
    const user = setupUser();
    const selectNode = vi.fn();
    unregisterCanvasNodeSearchProvider = registerCanvasNodeSearchProvider({
      searchNodes: (query) =>
        query.toLowerCase().includes("deploy")
          ? [
              {
                id: "node-1",
                label: "Deploy component",
                iconSlug: "box",
                keywords: ["deploy component", "node-1"],
              },
            ]
          : [],
      selectNode,
    });
    renderPalette("/org-1/apps/canvas-1");

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "deploy");
    await user.click(await screen.findByText("Deploy component"));

    expect(selectNode).toHaveBeenCalledWith("node-1");
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("Find apps, integrations, and commands...")).not.toBeInTheDocument();
    });
  });

  it("hides the invitation command without member create permission", async () => {
    permissionsState.permissions = defaultPermissions.filter(
      (permission) => permission.resource !== "members" || permission.action !== "create",
    );
    renderPalette();
    openPalette();
    expect(await screen.findByPlaceholderText("Find apps, integrations, and commands...")).toBeInTheDocument();
    expect(screen.queryByText("Invite by email")).not.toBeInTheDocument();
  });

  it("opens member settings to invite a specific recipient", async () => {
    const user = setupUser();
    renderPalette();
    openPalette();
    await user.click(await screen.findByText("Invite by email"));
    expect(navigateMock).toHaveBeenCalledWith("/org-1/settings/members");
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("Find apps, integrations, and commands...")).not.toBeInTheDocument();
    });
  });

  it("creates an app with the quick shortcut", async () => {
    renderPalette();

    openPalette();
    await waitFor(() => {
      expect(screen.getByText("New App").closest("[cmdk-item]")).not.toHaveAttribute("data-disabled", "true");
    });
    fireEvent.keyDown(document, { key: "/", metaKey: true });

    await waitFor(() => {
      expect(createCanvasMock).toHaveBeenCalledWith({ name: "generated-canvas", method: "ui" });
    });
    expect(navigateMock).toHaveBeenCalledWith("/org-1/apps/canvas-new");
  });

  it("collapses expanded section when back is clicked", async () => {
    const user = setupUser();
    renderPalette();

    openPalette();
    await user.click(await screen.findByText("Apps"));
    expect(await screen.findByText("Deploy API")).toBeInTheDocument();

    await user.click(screen.getByText("Back"));
    expect(screen.queryByText("Deploy API")).not.toBeInTheDocument();
    expect(screen.getByText("Apps")).toBeInTheDocument();
  });
});
