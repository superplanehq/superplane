import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import {
  LOW_CREDIT_USAGE_REPORT,
  PURCHASED_CREDIT_USAGE_REPORT,
  SPENT_CREDIT_USAGE_REPORT,
} from "../__fixtures__/usageReportFixtures";
import { HOSTED_CREDIT_RUNS_STOP_HINT } from "../lib/hostedCreditEmpty";

describe("LinesPage hosted credit banner", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("hides the banner when purchased hosted credit is comfortably above $20", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: PURCHASED_CREDIT_USAGE_REPORT,
        }}
      />,
    );

    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-empty-banner")).not.toBeInTheDocument();
  }, 10000);

  it("shows a trial-empty banner that opens Billing when welcome credit is spent", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
        }}
      />,
    );

    const banner = await screen.findByTestId("hosted-credit-empty-banner", {}, { timeout: 8000 });
    expect(banner).toHaveTextContent("Trial credit is empty");
    expect(banner).toHaveAttribute("data-tone", "warning");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      expect.stringContaining("/settings/organization/billing"),
    );
  }, 10000);

  it("shows a low-credit banner when purchased remaining credit is at or below $20", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: LOW_CREDIT_USAGE_REPORT,
        }}
      />,
    );

    const banner = await screen.findByTestId("hosted-credit-empty-banner", {}, { timeout: 8000 });
    expect(banner).toHaveTextContent("Hosted credit is low");
    expect(banner).toHaveTextContent("$15.00 remaining");
    expect(banner).toHaveTextContent(HOSTED_CREDIT_RUNS_STOP_HINT);
    expect(banner).toHaveAttribute("data-tone", "warning");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      expect.stringContaining("/settings/organization/billing"),
    );
  }, 10000);
});
