import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { catalogEntryToAutomation, catalogForColumn } from "../lib/columnAutomations";
import { useAddColumnAutomation } from "./useAddColumnAutomation";

const reviewRequestFeature = { enabled: false };
const installFactory = vi.fn();
const createAutomation = vi.fn();
const describeIntegration = vi.fn();
vi.mock("@/api-client", () => ({
  organizationsDescribeIntegration: (...args: unknown[]) => describeIntegration(...args),
}));
vi.mock("@/hooks/useFactoryData", () => ({
  useCreateFactoryAutomation: () => ({ mutateAsync: createAutomation, isPending: false }),
  factoryAppsKey: (organizationId: string, factoryId: string) => ["apps", organizationId, factoryId],
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (feature: string) => feature !== "factory_review_request" || reviewRequestFeature.enabled,
  }),
}));
vi.mock("@/pages/home/useInstallFactory", () => ({
  useInstallFactory: () => ({ installFactory, isInstalling: false }),
}));

beforeEach(() => {
  reviewRequestFeature.enabled = false;
  installFactory.mockReset();
  createAutomation.mockReset();
  createAutomation.mockResolvedValue({ id: "agent-app" });
  installFactory.mockResolvedValue({ canvasId: "closure-app" });
  describeIntegration.mockResolvedValue({ data: { integration: { metadata: { name: "workspace-vcs" } } } });
});

describe("add stage agent", () => {
  it.each(["phase-0", "phase-1"] as const)(
    "only blocks an agent installed in the target stage: %s",
    async (occupied) => {
      const agent = catalogForColumn(occupied).find((entry) => entry.kind === "agent-step")!;
      const { result } = renderHook(
        () =>
          useAddColumnAutomation({
            organizationId: "org",
            factoryId: "factory",
            factoryKey: "workspace",
            lineId: "line",
            appRepository: "acme/app",
            backlogRepository: "acme/app",
            defaultBranch: "main",
            githubIntegrationId: "vcs-integration",
            automationsFor: (key) =>
              key === occupied ? [catalogEntryToAutomation(agent, "Stage", "installed-agent")] : [],
          }),
        {
          wrapper: ({ children }: { children: ReactNode }) => (
            <QueryClientProvider client={new QueryClient()}>
              <MemoryRouter>{children}</MemoryRouter>
            </QueryClientProvider>
          ),
        },
      );
      act(() => result.current.openPicker("phase-1"));
      expect(result.current.takenIds.includes("agent-step")).toBe(occupied === "phase-1");
      await act(async () => result.current.onSelect(agent));
      expect(createAutomation.mock.calls.length).toBe(occupied === "phase-1" ? 0 : 1);
    },
  );
});

describe("add custom stage automation", () => {
  it("opens shared configuration with the custom stage on the workspace board", () => {
    const { result } = renderHook(
      () => ({
        automation: useAddColumnAutomation({
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "workspace",
          lineId: "line",
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          githubIntegrationId: "vcs-integration",
          automationsFor: () => [],
        }),
        location: useLocation(),
      }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={new QueryClient()}>
            <MemoryRouter>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );

    act(() => result.current.automation.openPicker("custom-stage-id", "Deploy to prod"));
    expect(result.current.location.pathname).toBe("/org/workspaces/workspace");
    expect(result.current.location.search).toBe("?customAutomation=new&automationColumn=custom-stage-id");
    expect(result.current.automation.pickerOpen).toBe(false);
  });
});

describe("add pull request closure", () => {
  for (const provider of ["github", "bitbucket"]) {
    it(`installs with ${provider} and stays on the board`, async () => {
      const client = new QueryClient();
      const invalidate = vi.spyOn(client, "invalidateQueries");
      const boardPath = "/org/workspaces/workspace";
      const { result } = renderHook(
        () => ({
          automation: useAddColumnAutomation({
            organizationId: "org",
            factoryId: "factory",
            factoryKey: "workspace",
            lineId: "line",
            appRepository: "acme/app",
            backlogRepository: "acme/app",
            defaultBranch: "main",
            githubIntegrationId: "vcs-integration",
            vcsProvider: provider,
            automationsFor: () => [],
          }),
          location: useLocation(),
        }),
        {
          wrapper: ({ children }: { children: ReactNode }) => (
            <QueryClientProvider client={client}>
              <MemoryRouter initialEntries={[boardPath]}>{children}</MemoryRouter>
            </QueryClientProvider>
          ),
        },
      );
      act(() => result.current.automation.openPicker("done"));
      const entry = result.current.automation.catalog.find((entry) => entry.kind === "pr-closure")!;
      await act(async () => result.current.automation.onSelect(entry));
      expect(installFactory).toHaveBeenCalledWith(
        expect.objectContaining({
          factoryId: "pr-closure",
          vcsProvider: provider,
          workspaceFactoryId: "factory",
          integrations: { [provider]: { id: "vcs-integration", name: "workspace-vcs", ready: true } },
          navigateOnComplete: false,
          startInitialRun: false,
        }),
      );
      expect(result.current.location.pathname).toBe(boardPath);
      expect(result.current.automation.pickerOpen).toBe(false);
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ["apps", "org", "factory"] });
    });
  }
});

describe("add visual evidence", () => {
  it("installs the GitHub canvas and stays on the board", async () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const boardPath = "/org/workspaces/workspace";
    const { result } = renderHook(
      () => ({
        automation: useAddColumnAutomation({
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "workspace",
          lineId: "line",
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          githubIntegrationId: "vcs-integration",
          vcsProvider: "github",
          automationsFor: () => [],
        }),
        location: useLocation(),
      }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={client}>
            <MemoryRouter initialEntries={[boardPath]}>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );
    act(() => result.current.automation.openPicker("verify"));
    const entry = result.current.automation.catalog.find((item) => item.kind === "visual-evidence")!;
    await act(async () => result.current.automation.onSelect(entry));
    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        factoryId: "visual-evidence",
        vcsProvider: "github",
        workspaceFactoryId: "factory",
        integrations: { github: { id: "vcs-integration", name: "workspace-vcs", ready: true } },
        navigateOnComplete: false,
        startInitialRun: false,
      }),
    );
    expect(result.current.location.pathname).toBe(boardPath);
    expect(result.current.automation.pickerOpen).toBe(false);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["apps", "org", "factory"] });
  });

  it("does not install visual evidence for Bitbucket", async () => {
    const { result } = renderHook(
      () =>
        useAddColumnAutomation({
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "workspace",
          lineId: "line",
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          githubIntegrationId: "vcs-integration",
          vcsProvider: "bitbucket",
          automationsFor: () => [],
        }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={new QueryClient()}>
            <MemoryRouter>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );
    act(() => result.current.openPicker("verify"));
    expect(result.current.catalog.some((item) => item.kind === "visual-evidence")).toBe(false);
    await act(async () =>
      result.current.onSelect({
        id: "visual-evidence",
        kind: "visual-evidence",
        name: "Visual Evidence",
        description: "Capture screenshots or a short video when a pull request changes the user interface.",
        trigger: "On pull request opened or updated",
        action: "Capture screenshots or a short video",
        iconSrc: "",
        iconAlt: "",
        unique: true,
      }),
    );
    expect(installFactory).not.toHaveBeenCalled();
    expect(result.current.pickerOpen).toBe(true);
  });
});

describe("add custom automation", () => {
  it("creates a distinct named automation in an alternate app stage", async () => {
    const entry = catalogForColumn("phase-1").find((item) => !item.unique)!;
    const { result } = renderHook(
      () => ({
        automation: useAddColumnAutomation({
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "workspace",
          lineId: "line",
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          githubIntegrationId: "vcs-integration",
          automationsFor: () => [catalogEntryToAutomation(entry, "Stage", "existing")],
        }),
        location: useLocation(),
      }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={new QueryClient()}>
            <MemoryRouter>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );
    act(() => result.current.automation.openPicker("phase-1"));
    await act(async () => result.current.automation.onSelect(entry));
    expect(result.current.automation.naming).toBe(true);
    createAutomation.mockResolvedValue({ id: "second-custom" });
    await act(async () => result.current.automation.createNamed("Second automation"));
    expect(createAutomation).toHaveBeenCalledWith({ name: "Second automation", columnKey: undefined });
    expect(result.current.location.pathname).toContain("second-custom");
  });

  it.each(["backlog", "phase-0", "verify", "done", "custom-stage-id"] as const)(
    "starts fresh shared setup with an existing custom automation in %s",
    async (column) => {
      const custom = catalogForColumn("verify").find((entry) => !entry.unique)!;
      const { result } = renderHook(
        () => ({
          automation: useAddColumnAutomation({
            organizationId: "org",
            factoryId: "factory",
            factoryKey: "workspace",
            lineId: "line",
            appRepository: "acme/app",
            backlogRepository: "acme/app",
            defaultBranch: "main",
            githubIntegrationId: "vcs-integration",
            automationsFor: (key) => (key === column ? [catalogEntryToAutomation(custom, column, "existing")] : []),
          }),
          location: useLocation(),
        }),
        {
          wrapper: ({ children }: { children: ReactNode }) => (
            <QueryClientProvider client={new QueryClient()}>
              <MemoryRouter>{children}</MemoryRouter>
            </QueryClientProvider>
          ),
        },
      );
      act(() => result.current.automation.openPicker(column));
      if (result.current.automation.pickerOpen) {
        const entry = result.current.automation.catalog.find((item) => !item.unique)!;
        await act(async () => result.current.automation.onSelect(entry));
      }
      const params = new URLSearchParams(result.current.location.search);
      expect(params.get("customAutomation")).toBe("new");
      expect(params.get("automationColumn")).toBe(
        column === "verify" ? null : column === "phase-0" ? "implement" : column,
      );
      expect(createAutomation).not.toHaveBeenCalled();
    },
  );

  it("opens the focused Verify setup route", async () => {
    const { result } = renderHook(
      () => ({
        automation: useAddColumnAutomation({
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "workspace",
          lineId: "line",
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          githubIntegrationId: "vcs-integration",
          vcsProvider: "bitbucket",
          automationsFor: () => [],
        }),
        location: useLocation(),
      }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={new QueryClient()}>
            <MemoryRouter initialEntries={["/org/workspaces/workspace"]}>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );

    act(() => result.current.automation.openPicker("verify"));
    const entry = result.current.automation.catalog.find((item) => item.kind === "custom-verification")!;
    await act(async () => result.current.automation.onSelect(entry));

    expect(result.current.location.pathname).toBe("/org/workspaces/workspace");
    expect(result.current.location.search).toBe("?customAutomation=new");
    expect(result.current.automation.naming).toBe(false);
  });

  it("opens focused setup when custom automation is the only available choice", () => {
    const configured = catalogForColumn("verify").map((entry) => catalogEntryToAutomation(entry, "Verify", entry.id));
    const { result } = renderHook(
      () => ({
        automation: useAddColumnAutomation({
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "workspace",
          lineId: "line",
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          githubIntegrationId: "vcs-integration",
          vcsProvider: "github",
          automationsFor: () => configured,
        }),
        location: useLocation(),
      }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={new QueryClient()}>
            <MemoryRouter initialEntries={["/org/workspaces/workspace"]}>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );

    act(() => result.current.automation.openPicker("verify"));

    expect(result.current.location.pathname).toBe("/org/workspaces/workspace");
    expect(result.current.location.search).toBe("?customAutomation=new");
    expect(result.current.automation.naming).toBe(false);
  });
});

describe("add Review request", () => {
  it.each([false, true])("gates discovery and installs with the flag enabled: %s", async (enabled) => {
    reviewRequestFeature.enabled = enabled;
    const { result } = renderHook(
      () =>
        useAddColumnAutomation({
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "workspace",
          lineId: "line",
          appRepository: "acme/app",
          backlogRepository: "acme/backlog",
          defaultBranch: "main",
          githubIntegrationId: "vcs-integration",
          vcsProvider: "github",
          automationsFor: () => [],
        }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={new QueryClient()}>
            <MemoryRouter>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );
    act(() => result.current.openPicker("verify"));
    const entry = result.current.catalog.find((item) => item.kind === "review-request");
    expect(Boolean(entry)).toBe(enabled);
    if (!entry) return;
    await act(async () => result.current.onSelect(entry));
    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        factoryId: "review-request",
        workspaceFactoryId: "factory",
        vcsProvider: "github",
        navigateOnComplete: false,
        startInitialRun: false,
      }),
    );
  });
});
