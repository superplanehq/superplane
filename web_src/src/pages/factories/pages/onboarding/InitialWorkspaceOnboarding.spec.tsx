import { render, screen, waitFor } from "@testing-library/react";
import { useEffect, useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { InitialWorkspaceOnboarding } from "./InitialWorkspaceOnboarding";
import { decideInitialWorkspaceOnboarding, type InitialWorkspaceLookup } from "./initialWorkspaceOnboardingScreen";
import { OnboardingWorkspaceResolutionProvider } from "./OnboardingWorkspaceResolutionProvider";

const workspace = { id: "factory-1", key: "NEWWO", name: "New workspace" };

const store = vi.hoisted(() => {
  const listeners = new Set<() => void>();
  return {
    reresolve: vi.fn<() => Promise<void>>().mockResolvedValue(undefined),
    publish() {
      for (const listener of listeners) listener();
    },
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    factories: {
      data: [] as Array<typeof workspace>,
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue({}),
    },
    factory: {
      data: undefined as typeof workspace | undefined,
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue({}),
    },
  };
});

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => store.factories,
  useFactory: (_organizationId: string, factoryId: string) =>
    factoryId
      ? store.factory
      : {
          data: undefined,
          isLoading: false,
          isFetching: false,
          isError: false,
          refetch: store.factory.refetch,
        },
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1" } }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ isLoading: false, canAct: () => true }),
}));

vi.mock("@/contexts/PermissionsProvider", () => ({
  PermissionsProvider: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/hooks/useRecordLastLocation", () => ({
  useRecordLastLocation: () => undefined,
}));

vi.mock("./OnboardingPage", () => ({
  OnboardingPage: () => <div data-testid="workspace-setup" />,
}));

function readyLookup(overrides: Partial<InitialWorkspaceLookup> = {}): InitialWorkspaceLookup {
  return {
    listError: false,
    resolution: "found",
    describeError: false,
    describeReady: true,
    workspaceOpen: false,
    reresolveRunning: false,
    reresolveAttempted: false,
    ...overrides,
  };
}

function Harness({
  organizationId = "dev-user",
  factoryKey = "NEWWO",
}: {
  organizationId?: string;
  factoryKey?: string;
}) {
  const [, setTick] = useState(0);
  useEffect(() => store.subscribe(() => setTick((tick) => tick + 1)), []);
  return (
    <MemoryRouter>
      <OnboardingWorkspaceResolutionProvider resolve={store.reresolve}>
        <InitialWorkspaceOnboarding organizationId={organizationId} factoryKey={factoryKey} />
      </OnboardingWorkspaceResolutionProvider>
    </MemoryRouter>
  );
}

describe("decideInitialWorkspaceOnboarding", () => {
  it("re-resolves a first list failure instead of showing the missing workspace", () => {
    expect(decideInitialWorkspaceOnboarding(readyLookup({ listError: true, describeReady: false }))).toEqual({
      phase: "reresolve",
      view: "loading",
    });
    expect(decideInitialWorkspaceOnboarding(readyLookup({ resolution: "not-found", describeReady: false }))).toEqual({
      phase: "reresolve",
      view: "loading",
    });
  });

  it("re-resolves a describe error", () => {
    expect(decideInitialWorkspaceOnboarding(readyLookup({ describeError: true, describeReady: false }))).toEqual({
      phase: "reresolve",
      view: "loading",
    });
  });

  it("keeps an open workspace on screen while a list refresh omits it and re-resolve is running", () => {
    expect(
      decideInitialWorkspaceOnboarding(
        readyLookup({ resolution: "not-found", describeReady: false, workspaceOpen: true }),
      ),
    ).toEqual({ phase: "reresolve", view: "setup" });
    expect(
      decideInitialWorkspaceOnboarding(
        readyLookup({
          resolution: "not-found",
          describeReady: false,
          workspaceOpen: true,
          reresolveRunning: true,
        }),
      ),
    ).toEqual({ phase: "reresolve", view: "setup" });
  });

  it("shows the missing workspace after one re-resolve still cannot open it", () => {
    expect(
      decideInitialWorkspaceOnboarding(
        readyLookup({ resolution: "not-found", describeReady: false, reresolveAttempted: true }),
      ),
    ).toEqual({ phase: "error", view: "error" });
    expect(
      decideInitialWorkspaceOnboarding(
        readyLookup({ describeError: true, describeReady: false, reresolveAttempted: true, workspaceOpen: true }),
      ),
    ).toEqual({ phase: "error", view: "error" });
  });

  it("lets the lookup proceed after the organization slug or workspace key changes", () => {
    expect(
      decideInitialWorkspaceOnboarding(
        readyLookup({
          resolution: "loading",
          describeReady: false,
          reresolveAttempted: true,
          reresolveRunning: false,
        }),
      ),
    ).toEqual({ phase: "loading", view: "loading" });
  });

  it("shows setup when the lookup finds the workspace", () => {
    expect(decideInitialWorkspaceOnboarding(readyLookup())).toEqual({ phase: "ready", view: "setup" });
    expect(decideInitialWorkspaceOnboarding(readyLookup({ reresolveRunning: true }))).toEqual({
      phase: "ready",
      view: "setup",
    });
  });
});

describe("InitialWorkspaceOnboarding", () => {
  beforeEach(() => {
    store.reresolve.mockReset();
    store.reresolve.mockResolvedValue(undefined);
    store.factories.data = [];
    store.factories.isLoading = false;
    store.factories.isFetching = false;
    store.factories.isError = false;
    store.factories.refetch.mockReset();
    store.factories.refetch.mockResolvedValue({});
    store.factory.data = undefined;
    store.factory.isLoading = false;
    store.factory.isFetching = false;
    store.factory.isError = false;
    store.factory.refetch.mockReset();
    store.factory.refetch.mockResolvedValue({});
  });

  it("shows loading, re-resolves once, and shows setup when the retry finds the workspace", async () => {
    store.factories.isError = true;
    store.factories.refetch.mockImplementation(async () => {
      store.factories.isError = false;
      store.factories.data = [workspace];
      store.factory.data = workspace;
      store.factory.isError = false;
    });

    render(<Harness />);

    expect(screen.getByTestId("workspace-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("factories-layout-error")).not.toBeInTheDocument();

    await waitFor(() => expect(store.reresolve).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId("workspace-setup")).toBeInTheDocument();
    expect(store.reresolve).toHaveBeenCalledTimes(1);
  });

  it("keeps setup on screen while re-resolve runs after a list refresh omits the open workspace", async () => {
    let release = () => undefined;
    store.reresolve.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          release = resolve;
        }),
    );
    store.factories.data = [workspace];
    store.factory.data = workspace;

    render(<Harness />);
    expect(await screen.findByTestId("workspace-setup")).toBeInTheDocument();

    store.factories.data = [];
    store.publish();

    await waitFor(() => expect(store.reresolve).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("workspace-setup")).toBeInTheDocument();
    expect(screen.queryByTestId("factories-layout-error")).not.toBeInTheDocument();

    release();
    expect(await screen.findByTestId("factories-layout-error")).toBeInTheDocument();
    expect(screen.queryByTestId("workspace-setup")).not.toBeInTheDocument();
    expect(store.reresolve).toHaveBeenCalledTimes(1);
  });

  it("shows loading and re-resolves once when describe fails before setup is open", async () => {
    store.factories.data = [workspace];
    store.factory.isError = true;
    store.factory.refetch.mockImplementation(async () => {
      store.factory.isError = false;
      store.factory.data = workspace;
    });

    render(<Harness />);

    expect(screen.getByTestId("workspace-loading")).toBeInTheDocument();
    await waitFor(() => expect(store.reresolve).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId("workspace-setup")).toBeInTheDocument();
  });
});
