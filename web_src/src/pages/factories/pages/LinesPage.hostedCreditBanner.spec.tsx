import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { LOW_CREDIT_USAGE_REPORT, SPENT_CREDIT_USAGE_REPORT } from "../__fixtures__/usageReportFixtures";

describe("LinesPage hosted credit banner", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("hides the banner when remaining hosted credit is comfortably above the low-credit threshold", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-empty-banner")).not.toBeInTheDocument();
  }, 10000);

  it("shows a red out-of-credit banner with a go-to-billing button when remaining credit is empty", async () => {
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
    expect(banner).toHaveTextContent("Hosted credit is empty");
    expect(banner).toHaveClass("border-red-200");
    expect(screen.queryByRole("link", { name: "View spending" })).not.toBeInTheDocument();

    const button = screen.getByRole("button", { name: "Go to billing" });
    expect(button).toBeInTheDocument();
  }, 10000);

  it("shows an amber low-credit banner when remaining credit is above zero but at or below $20", async () => {
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
    expect(banner).toHaveTextContent("Hosted credit is running low");
    expect(banner).toHaveClass("border-amber-200");
    expect(screen.getByRole("button", { name: "Go to billing" })).toBeInTheDocument();
  }, 10000);
});
