import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { HostedCreditEmptyBanner } from "./HostedCreditEmptyBanner";

describe("HostedCreditEmptyBanner", () => {
  it("links to billing with a view action when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled spendingHref="/org/workspaces/RF/settings/organization/billing" />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Hosted credit is empty");
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent(
      "SuperPlane-hosted runs cannot start. Open Billing to review remaining credit.",
    );
    expect(screen.getByRole("link", { name: "View billing" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/billing",
    );
  });

  it("links to billing with a view action when billing is off", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          billingEnabled={false}
          spendingHref="/org/workspaces/RF/settings/organization/billing"
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "View billing" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/billing",
    );
  });

  it("shows View billing for a non-owner when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          billingEnabled
          canManageBilling={false}
          spendingHref="/org/workspaces/RF/settings/organization/billing"
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "View billing" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/billing",
    );
    expect(screen.queryByRole("link", { name: "Add hosted credit" })).not.toBeInTheDocument();
  });
});
