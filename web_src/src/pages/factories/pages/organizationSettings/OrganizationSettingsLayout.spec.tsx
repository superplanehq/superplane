import { render, screen, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";

describe("legacy factory organization settings routes", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("keeps the workspace and opens Organization General", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/organization/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-organization-general")).toHaveAttribute(
      "aria-current",
      "page",
    );
  }, 10000);

  it("redirects the direct organization route to a workspace Organization Spending page", async () => {
    render(<FactoriesHarness pathSuffix="organization/llm-spend" factoriesFixture={defaultFactoriesFixture} />);

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-organization-spending")).toHaveAttribute(
      "aria-current",
      "page",
    );
  }, 10000);

  it("redirects workspace-usage into Organization Spending", async () => {
    render(<FactoriesHarness pathSuffix="organization/workspace-usage" factoriesFixture={defaultFactoriesFixture} />);

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-organization-spending")).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(
      await screen.findByText("Review factory token usage, VM time, and estimated spend for this organization."),
    ).toBeInTheDocument();
  }, 10000);

  it("redirects the old LLM spend URL into Organization Spending", async () => {
    render(
      <FactoriesHarness pathSuffix="settings/llm-spend?credit=added" factoriesFixture={defaultFactoriesFixture} />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-organization-spending")).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(
      await screen.findByText("Review factory token usage, VM time, and estimated spend for this organization."),
    ).toBeInTheDocument();
  }, 10000);
});
