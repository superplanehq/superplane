import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { HostedCreditEmptyBanner } from "./HostedCreditEmptyBanner";

const billingHref = "/org/workspaces/RF/settings/organization/billing";

describe("HostedCreditEmptyBanner", () => {
  it("links to billing with a view action when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Hosted credit is empty");
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent(
      "SuperPlane-hosted runs cannot start. Open Billing to review remaining credit.",
    );
    expect(screen.getByRole("link", { name: "View billing" })).toHaveAttribute("href", billingHref);
  });

  it("links to billing with a view action when billing is off", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled={false} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "View billing" })).toHaveAttribute("href", billingHref);
  });

  it("shows View billing for a non-owner when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled canManageBilling={false} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "View billing" })).toHaveAttribute("href", billingHref);
    expect(screen.queryByRole("link", { name: "Add hosted credit" })).not.toBeInTheDocument();
  });

  it("shows remaining trial credit and an Open billing action", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          billingEnabled
          kind="trial"
          remainingCreditCents={4124}
          welcomeCreditExpiresAt={new Date(Date.now() + 14 * 24 * 60 * 60 * 1000).toISOString()}
          spendingHref={billingHref}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Trial");
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("$41.24");
    expect(screen.getByRole("link", { name: "Open billing" })).toHaveAttribute("href", billingHref);
  });

  it("shows trial-empty copy", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled kind="trial-empty" spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Trial credit is empty");
    expect(screen.getByRole("link", { name: "Open billing" })).toHaveAttribute("href", billingHref);
  });

  it("shows trial-expired copy", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled kind="trial-expired" spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Trial ended");
    expect(screen.getByRole("link", { name: "Open billing" })).toHaveAttribute("href", billingHref);
  });
});
