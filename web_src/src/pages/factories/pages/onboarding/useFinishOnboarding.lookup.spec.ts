import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
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
import { CODEX_SUBSCRIPTION_PLAN, resolveOnboardingAgent } from "./onboardingAgentReadiness";

const refetchBilling = vi.fn<() => Promise<{ data: OrganizationsDescribeOrganizationBillingResponse; error: null }>>(
  async () => ({ data: { billingEnabled: false }, error: null }),
);
vi.mock("@/hooks/useOrganizationBilling", () => ({ useOrganizationBilling: () => ({ refetch: refetchBilling }) }));

let byokAllowed = true;
vi.mock("@/hooks/useBYOKAccess", () => ({ useBYOKAccess: () => ({ verify: async () => byokAllowed }) }));
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    refetch: async () => ({ data: { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" } }),
  }),
}));

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
    byokAllowed = true;
    refetchBilling.mockReset().mockResolvedValue({ data: { billingEnabled: false }, error: null });
    organizationsDescribeIntegration.mockReset();
    vi.mocked(showErrorToast).mockReset();
  });

  it.each(["pending", "connected", "hosted", "denied", "subscription"] as const)(
    "honors the model source when the Anthropic connection is %s",
    async (connection) => {
      byokAllowed = connection !== "denied" && connection !== "subscription";
      organizationsDescribeIntegration.mockResolvedValue({
        data: { integration: { metadata: { name: "github-acme" } } },
      });
      const createIntake = vi.fn().mockResolvedValue({ id: "intake-1" });
      const onProvisioned = vi.fn();
      const updateOnboarding = vi.fn().mockResolvedValue({});
      const installFactory = vi.fn().mockResolvedValue({ canvasId: "canvas-1", canvasName: "canvas-1" });
      const providerPlan = resolveOnboardingAgent({
        connected: new Set(connection === "pending" ? [] : ["claude"]),
        hostedModels: { anthropic: ["claude-sonnet-4-6", "claude-opus-5-5"], openai: [], openrouter: [] },
        defaultHostedProvider: "openrouter",
        defaultHostedModel: "openai/gpt-5",
        preferOwnKey: connection !== "hosted",
      });
      const plan = connection === "subscription" ? CODEX_SUBSCRIPTION_PLAN : providerPlan;
      const { result } = renderHook(
        () =>
          useFinishOnboarding({
            organizationId: "org-1",
            factoryId: "factory-1",
            factoryKey: "web",
            factory: null,
            setup: setup(),
            selections: {
              github: { id: "github-1", name: "github-acme", ready: true },
              ...(connection !== "pending"
                ? { claude: { id: "claude-1", name: "my-anthropic-key", ready: true } }
                : {}),
            },
            setSaving: vi.fn(),
            updateFactory: vi.fn().mockResolvedValue({}),
            updateOnboarding,
            installFactory,
            createLine: vi.fn().mockResolvedValue({ id: "line-1" }),
            listIntakes: vi.fn().mockResolvedValue([]),
            createIntake,
            updateIntake: vi.fn(),
            deleteIntake: vi.fn().mockResolvedValue({}),
            listApps: vi.fn().mockResolvedValue([]),
            resolveDefaultBranch: vi.fn().mockResolvedValue("main"),
            takenNames: [],
            remainingCreditCents: 5000,
            hostedModelsLoading: false,
            plan,
            onProvisioned,
          }),
        { wrapper },
      );

      await result.current();

      if (connection === "pending") {
        expect(installFactory).not.toHaveBeenCalled();
        expect(updateOnboarding).not.toHaveBeenCalled();
        expect(showErrorToast).toHaveBeenCalled();
        return;
      }
      expect(showErrorToast).not.toHaveBeenCalled();
      expect(updateOnboarding).toHaveBeenCalledWith(
        expect.objectContaining({
          agentHarness:
            connection === "connected"
              ? "AGENT_HARNESS_CLAUDE_CODE"
              : connection === "subscription"
                ? "AGENT_HARNESS_CODEX"
                : "AGENT_HARNESS_SUPERPLANE",
          agentIntegrationId: connection === "connected" ? "claude-1" : "",
          codexSubscription: connection === "subscription",
        }),
      );
      expect(installFactory).toHaveBeenCalled();
      if (connection === "denied") {
        expect(createIntake).toHaveBeenCalledWith({ source: "SOURCE_GITHUB_ISSUES" });
        expect(onProvisioned).not.toHaveBeenCalled();
      }
      for (const [input] of installFactory.mock.calls) {
        expect(input.agentRewrite).toEqual(
          connection === "connected"
            ? {
                component: "runnerClaudeCode",
                model: "claude-sonnet-4-6",
                planningModel: "claude-opus-5-5",
                credentials: { source: "integration", name: "my-anthropic-key" },
              }
            : connection === "subscription"
              ? {
                  component: "runnerCodex",
                  model: "",
                  planningModel: "",
                  credentials: { source: "subscription" },
                }
              : { component: "runnerSuperPlane", model: "", planningModel: "" },
        );
      }
    },
  );

  it.each(["disabled", "ready", "no-plan", "no-card", "unavailable"] as const)(
    "checks current billing before provisioning with billing %s",
    async (billing) => {
      if (billing === "unavailable") refetchBilling.mockRejectedValue(new Error("Billing unavailable"));
      else if (billing !== "disabled")
        refetchBilling.mockResolvedValue({
          data: {
            billingEnabled: true,
            planCategory: billing === "no-plan" ? "none" : "paid",
            requirePaymentMethod: true,
            hasPaymentMethod: billing !== "no-card",
          },
          error: null,
        });
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
            updateIntake: vi.fn(),
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

      if (["no-plan", "no-card", "unavailable"].includes(billing)) {
        expect(showErrorToast).toHaveBeenCalled();
        expect(updateOnboarding).not.toHaveBeenCalled();
        expect(installFactory).not.toHaveBeenCalled();
        return;
      }
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
    },
  );
});
