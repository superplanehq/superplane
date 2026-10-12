import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { unmockedSrc } from "@/test/unmockedModule";

const { organizationsDescribeIntegration } = vi.hoisted(() => ({
  organizationsDescribeIntegration: vi.fn(),
}));

vi.mock("@/api-client", () => {
  const actual = unmockedSrc<Record<string, unknown>>("api-client/index.ts");
  return { ...actual, organizationsDescribeIntegration };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

import { showErrorToast } from "@/lib/toast";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";
import { useFinishOnboarding } from "./useFinishOnboarding";

const readyPlan = {
  component: "runnerSuperPlane",
  credentialsSource: "hosted",
  harness: "AGENT_HARNESS_SUPERPLANE",
  model: "",
  planningModel: "",
} as const;

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return createElement(QueryClientProvider, { client: queryClient }, createElement(MemoryRouter, null, children));
}

function setup(): OnboardingSetupApi {
  return {
    selectedRepo: "acme/web",
    issuesRepo: "acme/web",
    workspaceName: "Web",
    issuesChoice: "vcs",
    vcsHost: "github",
  } as OnboardingSetupApi;
}

describe("useFinishOnboarding", () => {
  beforeEach(() => {
    organizationsDescribeIntegration.mockReset();
    vi.mocked(showErrorToast).mockReset();
  });

  it("resolves the installation name before it treats an unready saved GitHub selection as blocked", async () => {
    organizationsDescribeIntegration.mockResolvedValue({
      data: { integration: { metadata: { name: "github-acme" } } },
    });
    const updateOnboarding = vi.fn().mockResolvedValue({});
    const installFactory = vi.fn().mockResolvedValue({ canvasId: "canvas-1", canvasName: "canvas-1" });
    const onProvisioned = vi.fn();

    const { result } = renderHook(
      () =>
        useFinishOnboarding({
          organizationId: "org-1",
          factoryId: "factory-1",
          factoryKey: "web",
          factory: null,
          setup: setup(),
          selections: { github: { id: "int-1", name: "int-1", ready: false } },
          setSaving: vi.fn(),
          updateFactory: vi.fn().mockResolvedValue({}),
          updateOnboarding,
          installFactory,
          createLine: vi.fn().mockResolvedValue({ id: "line-1" }),
          listIntakes: vi.fn().mockResolvedValue([]),
          createIntake: vi.fn().mockResolvedValue({ id: "intake-1" }),
          deleteIntake: vi.fn().mockResolvedValue({}),
          listApps: vi.fn().mockResolvedValue([]),
          resolveDefaultBranch: vi.fn().mockResolvedValue("main"),
          takenNames: [],
          remainingCreditCents: 5000,
          hostedModelsLoading: false,
          plan: readyPlan,
          onProvisioned,
        }),
      { wrapper },
    );

    await result.current();

    expect(organizationsDescribeIntegration).toHaveBeenCalled();
    expect(showErrorToast).not.toHaveBeenCalled();
    expect(updateOnboarding).toHaveBeenCalled();
    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        integrations: expect.objectContaining({
          github: { id: "int-1", name: "github-acme", ready: true },
        }),
      }),
    );
  });
});
