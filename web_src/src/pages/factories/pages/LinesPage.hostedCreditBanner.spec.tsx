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
  BUSINESS_ORGANIZATION_BILLING,
  LAPSED_ORGANIZATION_BILLING,
  LAPSED_TOPUP_USAGE_REPORT,
  LOW_CREDIT_USAGE_REPORT,
  PURCHASED_CREDIT_USAGE_REPORT,
  SPENT_CREDIT_USAGE_REPORT,
} from "../__fixtures__/usageReportFixtures";
import { HOSTED_CREDIT_RUNS_STOP_HINT, welcomeCreditHeaderLabel } from "../lib/hostedCreditEmpty";

const defaultTrialLabel = welcomeCreditHeaderLabel(new Date("2026-09-22T12:00:00.000Z"));

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
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
        }}
      />,
    );

    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-empty-banner")).not.toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-header-kicker")).not.toBeInTheDocument();
  }, 10000);

  it("shows the trial chip next to the line title when welcome credit remains", async () => {
    render(<FactoriesHarness pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`} />);

    const kicker = await screen.findByTestId("hosted-credit-header-kicker", {}, { timeout: 8000 });
    expect(kicker).toHaveTextContent(defaultTrialLabel);
    expect(kicker).toHaveTextContent("$41.24");
    expect(screen.getByTestId("workspace-page-header-title").parentElement).toContainElement(kicker);
    expect(screen.queryByTestId("workspace-page-header-above-title")).not.toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-empty-banner")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute(
      "href",
      expect.stringContaining("/settings/organization/billing"),
    );
  }, 10000);

  it("keeps the trial chip next to the line title when welcome credit is spent", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
        }}
      />,
    );

    const kicker = await screen.findByTestId("hosted-credit-header-kicker", {}, { timeout: 8000 });
    expect(kicker).toHaveAttribute("data-kind", "trial");
    expect(kicker).toHaveTextContent(defaultTrialLabel);
    expect(kicker).toHaveTextContent("$0.00");
    expect(screen.getByTestId("workspace-page-header-title").parentElement).toContainElement(kicker);
    expect(screen.queryByTestId("hosted-credit-empty-banner")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute(
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
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
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

  it("shows a no-plan chip next to the line title when the plan is none", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: LAPSED_TOPUP_USAGE_REPORT,
          organizationBilling: LAPSED_ORGANIZATION_BILLING,
        }}
      />,
    );

    const kicker = await screen.findByTestId("hosted-credit-header-kicker", {}, { timeout: 8000 });
    expect(kicker).toHaveAttribute("data-kind", "lapsed");
    expect(kicker).toHaveTextContent("No plan");
    expect(screen.getByTestId("workspace-page-header-title").parentElement).toContainElement(kicker);
    expect(screen.queryByTestId("hosted-credit-empty-banner")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute(
      "href",
      expect.stringContaining("/settings/organization/billing"),
    );
  }, 10000);
});
