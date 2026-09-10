import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  defaultFactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";
import { DEFAULT_USAGE_HISTORY_ROWS, usageHistoryRows } from "../../__fixtures__/usageHistoryFixtures";
import { workOrderDetailPath } from "../../lib/factoryPagePaths";

describe("OrganizationSettingsUsagePage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows one row per task run with tokens, VM time, cost, and model or machine", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/usage`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByRole("heading", { name: "Usage" })).toBeInTheDocument();
    expect(screen.getByText("Review task spend for this workspace.")).toBeInTheDocument();
    expect(
      screen.getByText("This list does not show which credit grant paid. Your keys spend is estimated."),
    ).toBeInTheDocument();

    const table = await screen.findByTestId("organization-usage-history", {}, { timeout: 8000 });
    expect(within(table).getByRole("columnheader", { name: "Date" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "User" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Task" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Model" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Tokens" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Token price" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "VM type" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Time" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "VM price" })).toBeInTheDocument();

    const rows = within(table).getAllByTestId("organization-usage-row");
    expect(rows).toHaveLength(3);

    expect(rows[0]).toHaveTextContent("RF-101");
    expect(rows[0]).not.toHaveTextContent("Reconcile duplicate refunds in ledger");
    expect(rows[0]).toHaveTextContent("22k");
    expect(rows[0]).toHaveTextContent("1 min 30 s");
    expect(rows[0]).toHaveTextContent("$0.03");
    expect(rows[0]).toHaveTextContent("$0.50");
    expect(rows[0]).toHaveTextContent("anthropic/claude-sonnet-4-6 (your keys)");
    expect(rows[0]).toHaveTextContent("e1-large-amd64");
    expect(rows[0]).toHaveTextContent("Leonardo DiCaprio");
    expect(within(rows[0]).getByRole("link", { name: "RF-101" })).toHaveAttribute(
      "href",
      workOrderDetailPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "101"),
    );

    expect(rows[1]).toHaveTextContent("RF-103");
    expect(rows[1]).not.toHaveTextContent("Add refund reconciliation test");
    expect(rows[1]).toHaveTextContent("1.8k");
    expect(rows[1]).toHaveTextContent("12 s");
    expect(rows[1]).toHaveTextContent("$1.50");
    expect(rows[1]).toHaveTextContent("$0.25");
    expect(rows[1]).toHaveTextContent("anthropic/claude-sonnet-4-6");
    expect(rows[1]).toHaveTextContent("e1-standard-amd64");
    expect(rows[1]).not.toHaveTextContent("your keys");
    expect(rows[1]).toHaveTextContent("Arnold Schwarzenegger");

    expect(rows[2]).toHaveTextContent("anthropic/claude-haiku-4-5 · anthropic/claude-sonnet-4-6 (your keys)");
    expect(rows[2]).toHaveTextContent("e1-large-amd64");
    expect(rows[2]).toHaveTextContent("$1.23");
    expect(rows[2]).toHaveTextContent("$0.17");
  }, 10000);

  it("shows an empty state when the period has no task spend", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/usage`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          usageHistoryByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
        }}
      />,
    );

    expect(await screen.findByText("No task spend in this period.")).toBeInTheDocument();
    expect(screen.queryByTestId("organization-usage-row")).not.toBeInTheDocument();
  }, 10000);

  it("puts Export CSV next to the period control and downloads all usage rows", async () => {
    const user = userEvent.setup();
    const capturedBlobs: Blob[] = [];
    const createObjectURL = vi.spyOn(URL, "createObjectURL").mockImplementation((blob) => {
      capturedBlobs.push(blob as Blob);
      return "blob:mock-usage-csv";
    });
    const revokeObjectURL = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});

    try {
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/usage`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );
      await screen.findByTestId("organization-usage-history", {}, { timeout: 8000 });

      const actions = screen.getByTestId("workspace-page-header-actions");
      const buttons = within(actions).getAllByRole("button");
      expect(buttons[0]).toHaveTextContent("Export CSV");
      expect(buttons[0]).toHaveAccessibleName("Export all usage as CSV");
      expect(buttons[1]).toHaveAttribute("data-testid", "usage-period");

      await user.click(buttons[0]);

      await waitFor(() => expect(capturedBlobs).toHaveLength(1));
      expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock-usage-csv");

      const csv = await capturedBlobs[0].text();
      expect(csv).toContain("Date,User,Task,Model,Tokens,Token price,VM type,Time,VM price");
      for (const row of DEFAULT_USAGE_HISTORY_ROWS) {
        expect(csv).toContain(row.workOrderKey);
      }

      await screen.findByText("Export CSV");
    } finally {
      createObjectURL.mockRestore();
      revokeObjectURL.mockRestore();
    }
  }, 10000);

  it("pages through more than 50 task runs", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/usage`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          usageHistoryByFactoryId: { [PRIMARY_FACTORY_ID]: usageHistoryRows(51) },
        }}
      />,
    );

    const pagination = await screen.findByTestId("organization-usage-pagination", {}, { timeout: 8000 });
    expect(pagination).toHaveTextContent("Showing 1–50 of 51");
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByText("Showing 51–51 of 51")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
    expect(screen.getAllByTestId("organization-usage-row")).toHaveLength(1);
  }, 10000);
});
