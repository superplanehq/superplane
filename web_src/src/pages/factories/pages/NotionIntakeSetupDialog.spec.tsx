import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { NotionIntakeSetupDialog } from "./NotionIntakeSetupDialog";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({
    data: [
      {
        metadata: { id: "integration-1", name: "Notion", integrationName: "notion" },
        status: { state: "ready" },
      },
    ],
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({ data: [{ name: "notion", label: "Notion" }] }),
  useCreateIntegration: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
  useIntegrationResources: () => ({
    data: [
      { id: "db-1", name: "Tasks" },
      { id: "db-2", name: "Bugs" },
    ],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({ open, onCreated }: { open: boolean; onCreated: (id: string) => void }) =>
    open ? (
      <button type="button" data-testid="finish-connect" onClick={() => onCreated("integration-new")}>
        Finish connect
      </button>
    ) : null,
}));

describe("NotionIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
  });

  it("finishes setup once a connection and a database are chosen", async () => {
    const user = userEvent.setup();
    render(<NotionIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    expect(screen.getByRole("heading", { name: "Connect Notion" })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("button", { name: "Select database" })).toBeEnabled());
    await user.click(screen.getByRole("button", { name: "Select database" }));

    expect(screen.getByRole("heading", { name: "Choose a database" })).toBeInTheDocument();
    await user.click(screen.getByTestId("notion-database-db-1"));
    await user.click(screen.getByRole("button", { name: "Finish setup" }));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_NOTION_PAGES",
        integrationId: "integration-1",
        resourceId: "db-1",
      });
    });

    // The backend seeds the newest pages, so the wizard never asks the user
    // to pick pages by hand.
    expect(screen.getByRole("heading", { name: "Setup complete" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "SuperPlane is adding the newest pages of the database to the Backlog. SuperPlane checks the database every minute, so later pages arrive shortly after your team adds them.",
      ),
    ).toBeInTheDocument();
  });

  it("filters the database list by name", async () => {
    const user = userEvent.setup();
    render(<NotionIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Select database" }));
    await user.type(screen.getByLabelText("Search databases"), "bug");

    expect(screen.queryByTestId("notion-database-db-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("notion-database-db-2")).toBeInTheDocument();
  });

  it("opens database selection as soon as a new account is connected", async () => {
    const user = userEvent.setup();
    render(<NotionIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Connect another account" }));
    await user.click(screen.getByTestId("finish-connect"));

    expect(screen.getByRole("heading", { name: "Choose a database" })).toBeInTheDocument();
    expect(screen.getByTestId("notion-database-db-1")).toBeInTheDocument();
  });
});
