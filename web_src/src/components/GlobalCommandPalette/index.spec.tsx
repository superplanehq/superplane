import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router-dom";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { GlobalCommandPalette } from ".";
import { openGlobalCommandPalette } from "./controller";

const { accountState, createCanvasMock, defaultAccount, defaultPermissions, navigateMock, permissionsState } =
  vi.hoisted(() => {
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
  useOrganizationInviteLink: () => ({
    data: { token: "test-invite-token", enabled: true },
  }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({
    data: [
      {
        metadata: { id: "int-1", name: "puppies-github", integrationName: "github" },
        status: { state: "ready" },
      },
    ],
  }),
}));

vi.mock("@/hooks/useServiceAccounts", () => ({
  useServiceAccounts: () => ({
    data: [{ id: "sa-1", name: "deploy-bot" }],
  }),
}));

vi.mock("@/lib/canvasNameGenerator", () => ({
  generateCanvasName: () => "generated-canvas",
}));

function openPalette() {
  openGlobalCommandPalette();
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

describe("GlobalCommandPalette", () => {
  beforeEach(() => {
    accountState.account = defaultAccount;
    accountState.loading = false;
    Element.prototype.scrollIntoView = vi.fn();
    createCanvasMock.mockReset();
    createCanvasMock.mockResolvedValue({ data: { canvas: { metadata: { id: "canvas-new" } } } });
    navigateMock.mockReset();
    permissionsState.permissions = [...defaultPermissions];
  });

  it("opens and shows quick links", async () => {
    renderPalette();

    openPalette();

    expect(await screen.findByPlaceholderText("Find apps, integrations, and commands...")).toBeInTheDocument();
    expect(screen.getByText("New App")).toBeInTheDocument();
    expect(screen.getByText("Copy Invite Link")).toBeInTheDocument();
    expect(screen.getByText("Apps")).toBeInTheDocument();
    expect(screen.getByText("Integrations")).toBeInTheDocument();
    expect(screen.getByText("Go to Docs")).toBeInTheDocument();
    expect(screen.getByText("Sign Out")).toBeInTheDocument();
  });

  it("closes with CMD+K while the command input is focused", async () => {
    renderPalette();

    openPalette();
    const input = await screen.findByPlaceholderText("Find apps, integrations, and commands...");
    input.focus();
    fireEvent.keyDown(input, { key: "k", metaKey: true });

    await waitFor(() => {
      expect(screen.queryByPlaceholderText("Find apps, integrations, and commands...")).not.toBeInTheDocument();
    });
  });

  it("does not open before the account is available", () => {
    accountState.account = null;
    renderPalette("/login");

    openPalette();

    expect(screen.queryByPlaceholderText("Find apps, integrations, and commands...")).not.toBeInTheDocument();
  });

  it("expands app list when clicking Apps", async () => {
    const user = userEvent.setup();
    renderPalette();

    openPalette();
    await user.click(await screen.findByText("Apps"));

    expect(await screen.findByText("Deploy API")).toBeInTheDocument();
    expect(screen.getByText("Database Backups")).toBeInTheDocument();
  });

  it("navigates to an app when selected from expanded list", async () => {
    const user = userEvent.setup();
    renderPalette();

    openPalette();
    await user.click(await screen.findByText("Apps"));
    await user.click(await screen.findByText("Database Backups"));

    expect(navigateMock).toHaveBeenCalledWith("/org-1/apps/canvas-2");
  });

  it("searches apps by name", async () => {
    const user = userEvent.setup();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "Deploy");

    expect(await screen.findByText("Deploy API")).toBeInTheDocument();
  });

  it("searches integrations by name", async () => {
    const user = userEvent.setup();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "puppies");

    expect(await screen.findByText("puppies-github")).toBeInTheDocument();
  });

  it("searches service accounts by name", async () => {
    const user = userEvent.setup();
    renderPalette();

    openPalette();
    await user.type(await screen.findByPlaceholderText("Find apps, integrations, and commands..."), "deploy-bot");

    expect(await screen.findByText("deploy-bot")).toBeInTheDocument();
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
    const user = userEvent.setup();
    renderPalette();

    openPalette();
    await user.click(await screen.findByText("Apps"));
    expect(await screen.findByText("Deploy API")).toBeInTheDocument();

    await user.click(screen.getByText("Back"));
    expect(screen.queryByText("Deploy API")).not.toBeInTheDocument();
    expect(screen.getByText("Apps")).toBeInTheDocument();
  });
});
