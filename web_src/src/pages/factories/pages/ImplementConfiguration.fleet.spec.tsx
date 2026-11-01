import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "bun:test";
import type { SuperplaneComponentsNode } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { ImplementConfiguration } from "./ImplementConfiguration";

vi.mock("@/hooks/useComponentData", () => ({ useComponent: () => ({ data: undefined }) }));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => false }) }));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({ data: [] }) }));
vi.mock("@/hooks/useOrganizationRunnerFleets", () => ({
  useOrganizationRunnerFleets: () => ({
    data: {
      fleets: [
        { id: "team-linux", scope: "organization" },
        { id: "e1-large-amd64", scope: "installation" },
      ],
    },
    isLoading: false,
    isError: false,
  }),
}));

it("saves an implementation fleet only on explicit Save and retains it on reopen", async () => {
  const onSave = vi.fn();
  const node: SuperplaneComponentsNode = {
    id: "implementation-agent-no-issue",
    name: "Implement",
    component: "runnerClaudeCode",
    concurrency: { max: 3 },
    configuration: { machineType: "e1-large-amd64", model: "sonnet", executionTimeoutSeconds: 3600, steps: [] },
  };
  const editor = (current: SuperplaneComponentsNode) => (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <ImplementConfiguration agent={current} organizationId="org-1" onSave={onSave} />
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(editor(node));
  await userEvent.click(within(screen.getByRole("group", { name: "Runner fleet" })).getByRole("combobox"));
  await userEvent.click(screen.getByRole("option", { name: "team-linux" }));
  expect(onSave).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
  expect(onSave).toHaveBeenCalledWith(
    expect.objectContaining({
      configuration: { ...node.configuration, machineType: "team-linux" },
      concurrency: node.concurrency,
    }),
  );
  view.unmount();
  render(editor({ ...node, configuration: { ...node.configuration, machineType: "team-linux" } }));
  expect(within(screen.getByRole("group", { name: "Runner fleet" })).getByRole("combobox")).toHaveTextContent(
    "team-linux",
  );
  expect(onSave).toHaveBeenCalledTimes(1);
});
