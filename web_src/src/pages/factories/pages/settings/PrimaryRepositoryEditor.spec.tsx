import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it } from "bun:test";

import { integrationKeys } from "@/hooks/useIntegrations";
import { PrimaryRepositoryRow } from "./PrimaryRepositoryEditor";

it("discards primary repository edits when switching workspaces with the same repository", async () => {
  const user = userEvent.setup();
  const client = new QueryClient();
  const props = {
    organizationId: "organization",
    integrationId: "github",
    repository: "acme/api",
    branch: "main",
    host: "github" as const,
    bitbucket: false,
    repositories: ["acme/api"],
    catalogRepositories: [{ fullName: "acme/api", defaultBranch: "main" }],
    repositoriesLoading: false,
    repositoriesError: false,
    canUpdate: true,
  };
  client.setQueryData(integrationKeys.resources("organization", "github", "branch", { repository: "acme/api" }), [
    { name: "main" },
    { name: "develop" },
  ]);
  client.setQueryData(
    [...integrationKeys.resources("organization", "github", "default_branch", { repository: "acme/api" }), "name"],
    "main",
  );
  const row = (factoryId: string) => (
    <QueryClientProvider client={client}>
      <ul>
        <PrimaryRepositoryRow {...props} factoryId={factoryId} />
      </ul>
    </QueryClientProvider>
  );
  const view = render(row("first-workspace"));
  await user.click(screen.getByTestId("agent-cross-repo-primary-toggle"));
  await user.click(screen.getByRole("combobox", { name: "Default branch" }));
  await user.click(await screen.findByRole("option", { name: "develop" }));
  expect(screen.getByTestId("agent-primary-save")).toBeEnabled();

  view.rerender(row("second-workspace"));

  expect(screen.getByRole("combobox", { name: "Default branch" })).toHaveTextContent("main");
  expect(screen.getByTestId("agent-primary-save")).toBeDisabled();
});
