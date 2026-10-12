import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type * as ReactRouterModule from "react-router";

import { factoryAppsKey, factoryQueryKeys } from "@/hooks/useFactoryData";
import { unmockedPackage } from "@/test/unmockedModule";

import { useInstallFactory, type InstallFactoryInput } from "./useInstallFactory";

const ORGANIZATION_ID = "org-1";
const WORKSPACE_FACTORY_ID = "factory-1";

const ensureFactoryCanvas = vi.hoisted(() =>
  vi.fn(async () => ({ canvasId: "canvas-1", canvasName: "Merge confidence" })),
);
const materializeAndCommitFactoryTemplate = vi.hoisted(() => vi.fn(async () => undefined));
const invokeFactoryRun = vi.hoisted(() => vi.fn(async () => undefined));
const navigate = vi.hoisted(() => vi.fn());

vi.mock("./installFactoryCanvas", () => ({
  ensureFactoryCanvas,
  materializeAndCommitFactoryTemplate,
  invokeFactoryRun,
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true }),
}));

vi.mock("react-router", () => {
  const actual = unmockedPackage<typeof ReactRouterModule>("react-router/dist/development/index.js");
  return {
    ...actual,
    useNavigate: () => navigate,
    useParams: () => ({ organizationId: ORGANIZATION_ID }),
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

function workspaceAppsKey() {
  return factoryAppsKey(ORGANIZATION_ID, WORKSPACE_FACTORY_ID);
}

function isWorkspaceAppsQuery(filters: unknown) {
  return Boolean(
    filters && typeof filters === "object" && "queryKey" in filters && sameKey(filters.queryKey, workspaceAppsKey()),
  );
}

function isWorkOrdersQuery(filters: unknown) {
  const workOrdersKey = factoryQueryKeys.workOrders(ORGANIZATION_ID, WORKSPACE_FACTORY_ID);
  return Boolean(
    filters && typeof filters === "object" && "queryKey" in filters && sameKey(filters.queryKey, workOrdersKey),
  );
}

function sameKey(actual: unknown, expected: readonly unknown[]) {
  return (
    Array.isArray(actual) &&
    actual.length === expected.length &&
    actual.every((part, index) => part === expected[index])
  );
}

function installInput(workspaceFactoryId?: string): InstallFactoryInput {
  return {
    factoryId: "risk-score",
    workspaceFactoryId,
    integrations: {},
    installParams: {},
    startingTaskPrompt: "",
    navigateOnComplete: false,
    startInitialRun: false,
  };
}

function renderInstaller(queryClient: QueryClient) {
  return renderHook(() => useInstallFactory({ organizationId: ORGANIZATION_ID }), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children),
  });
}

describe("useInstallFactory", () => {
  beforeEach(() => {
    ensureFactoryCanvas.mockClear();
    materializeAndCommitFactoryTemplate.mockReset();
    materializeAndCommitFactoryTemplate.mockResolvedValue(undefined);
    invokeFactoryRun.mockClear();
    navigate.mockClear();
  });

  it("refreshes workspace automations before a workspace install resolves", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    let releaseRefresh = () => {};
    const refresh = new Promise<void>((resolve) => {
      releaseRefresh = resolve;
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries").mockImplementation((filters) => {
      if (isWorkspaceAppsQuery(filters)) {
        return refresh;
      }
      return Promise.resolve();
    });
    const { result } = renderInstaller(queryClient);

    let settled = false;
    const pending = result.current.installFactory(installInput(WORKSPACE_FACTORY_ID))?.then(() => {
      settled = true;
    });

    await waitFor(() => {
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: workspaceAppsKey() });
    });
    expect(settled).toBe(false);
    expect(invalidateQueries.mock.calls.some(([filters]) => isWorkOrdersQuery(filters))).toBe(false);

    releaseRefresh();
    await pending;
    expect(settled).toBe(true);
  });

  it("leaves workspace automations cached when the install has no workspace", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderInstaller(queryClient);

    await result.current.installFactory(installInput());

    expect(invalidateQueries.mock.calls.some(([filters]) => isWorkspaceAppsQuery(filters))).toBe(false);
  });

  it("leaves workspace automations cached when materialize fails", async () => {
    materializeAndCommitFactoryTemplate.mockRejectedValueOnce(new Error("materialize failed"));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderInstaller(queryClient);

    await expect(result.current.installFactory(installInput(WORKSPACE_FACTORY_ID))).rejects.toThrow(
      "materialize failed",
    );

    expect(invalidateQueries.mock.calls.some(([filters]) => isWorkspaceAppsQuery(filters))).toBe(false);
  });
});
