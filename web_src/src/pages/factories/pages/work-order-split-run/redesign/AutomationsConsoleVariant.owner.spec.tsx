import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { STORYBOOK_ME_USER_ID } from "../../../__fixtures__/factoryPageIds";
import { LiveHeaderSpendProvider } from "../liveHeaderSpendContext";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [
      {
        metadata: { id: "storybook-user", email: "john.doe@superplane.dev" },
        spec: { displayName: "Leonardo DiCaprio" },
      },
      {
        metadata: { id: "user-me", email: "casey@example.com" },
        spec: { displayName: "Casey Reviewer" },
      },
    ],
    isLoading: false,
  }),
}));

function OwnerEditHost({
  initialIds,
  initialOwner,
  canEditOwner = true,
  onOwnerSave,
}: {
  initialIds: string[];
  initialOwner: (typeof SPLIT_RUN_RUNNING)["owner"];
  canEditOwner?: boolean;
  onOwnerSave: (assigneeIds: string[]) => Promise<void>;
}) {
  const [assigneeIds, setAssigneeIds] = useState(initialIds);
  const [owner, setOwner] = useState(initialOwner);
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <LiveHeaderSpendProvider>
              <AutomationsConsoleVariant
                fixture={SPLIT_RUN_RUNNING}
                source={SPLIT_RUN_RUNNING.source}
                organizationId="org-1"
                assigneeIds={assigneeIds}
                owner={owner}
                canEditOwner={canEditOwner}
                onOwnerSave={async (nextIds) => {
                  await onOwnerSave(nextIds);
                  setAssigneeIds(nextIds);
                  setOwner(
                    nextIds[0] === "user-me"
                      ? { id: "user-me", name: "Casey Reviewer", initials: "CR" }
                      : { id: "", name: "", initials: "" },
                  );
                }}
              />
            </LiveHeaderSpendProvider>
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("AutomationsConsoleVariant owner", () => {
  it("saves the selected person as the only owner and shows that person", async () => {
    const onOwnerSave = vi.fn().mockResolvedValue(undefined);
    render(
      <OwnerEditHost
        initialIds={[STORYBOOK_ME_USER_ID]}
        initialOwner={SPLIT_RUN_RUNNING.owner}
        onOwnerSave={onOwnerSave}
      />,
    );

    fireEvent.click(screen.getByTestId("task-edit-owner"));
    fireEvent.click(screen.getByRole("option", { name: "Casey Reviewer" }));

    await waitFor(() => expect(onOwnerSave).toHaveBeenCalledWith(["user-me"]));
    expect(screen.getByRole("button", { name: "Owner: Casey Reviewer" })).toBeInTheDocument();
  });

  it("clears the owner and shows the assign label", async () => {
    const onOwnerSave = vi.fn().mockResolvedValue(undefined);
    render(
      <OwnerEditHost
        initialIds={[STORYBOOK_ME_USER_ID]}
        initialOwner={SPLIT_RUN_RUNNING.owner}
        onOwnerSave={onOwnerSave}
      />,
    );

    fireEvent.click(screen.getByTestId("task-edit-owner"));
    fireEvent.click(screen.getByRole("option", { name: "No owner" }));

    await waitFor(() => expect(onOwnerSave).toHaveBeenCalledWith([]));
    expect(screen.getByRole("button", { name: "Assign owner" })).toBeInTheDocument();
    expect(screen.queryByText(SPLIT_RUN_RUNNING.owner.name)).not.toBeInTheDocument();
  });

  it("shows no owner as text when edit is not allowed", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const owner = { id: "author", name: "Task Author", initials: "TA" };
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <LiveHeaderSpendProvider>
                <AutomationsConsoleVariant
                  fixture={{ ...SPLIT_RUN_RUNNING, owner }}
                  source={SPLIT_RUN_RUNNING.source}
                  organizationId="org-1"
                  assigneeIds={[]}
                  canEditOwner={false}
                  owner={owner}
                />
              </LiveHeaderSpendProvider>
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const panel = screen.getByTestId("redesign-console-summary");
    expect(within(panel).queryByTestId("task-edit-owner")).not.toBeInTheDocument();
    expect(within(panel).getByTestId("empty-owner-mark")).toBeInTheDocument();
    expect(within(panel).queryByText("Task Author")).not.toBeInTheDocument();
  });
});
