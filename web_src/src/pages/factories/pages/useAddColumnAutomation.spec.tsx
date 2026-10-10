import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { useAddColumnAutomation } from "./useAddColumnAutomation";

const installFactory = vi.fn();
const describeIntegration = vi.fn();
vi.mock("@/api-client", () => ({
  organizationsDescribeIntegration: (...args: unknown[]) => describeIntegration(...args),
}));
vi.mock("@/hooks/useFactoryData", () => ({
  useCreateFactoryAutomation: () => ({ isPending: false }),
  factoryAppsKey: (organizationId: string, factoryId: string) => ["apps", organizationId, factoryId],
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => true }) }));
vi.mock("@/pages/home/useInstallFactory", () => ({
  useInstallFactory: () => ({ installFactory, isInstalling: false }),
}));

beforeEach(() => {
  installFactory.mockReset();
  installFactory.mockResolvedValue({ canvasId: "closure-app" });
  describeIntegration.mockResolvedValue({ data: { integration: { metadata: { name: "workspace-vcs" } } } });
});

describe("add pull request closure", () => {
  for (const provider of ["github", "bitbucket"]) {
    it(`installs with ${provider} and stays on the board`, async () => {
      const client = new QueryClient();
      const invalidate = vi.spyOn(client, "invalidateQueries");
      const boardPath = "/org/workspaces/workspace/lines/line";
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
