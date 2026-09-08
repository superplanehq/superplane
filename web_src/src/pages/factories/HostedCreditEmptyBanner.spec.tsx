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
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("SuperPlane-hosted runs cannot start.");
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveAttribute("data-tone", "warning");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });

  it("links to billing with a view action when billing is off", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled={false} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });

  it("shows Add credits for a non-owner when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled canManageBilling={false} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
    expect(screen.queryByRole("link", { name: "Add hosted credit" })).not.toBeInTheDocument();
  });

  it("shows remaining trial credit and an Add credits action", () => {
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

    const banner = screen.getByTestId("hosted-credit-empty-banner");
    expect(banner).toHaveTextContent("Trial");
    expect(banner).toHaveTextContent("$41.24 remaining");
    expect(banner).toHaveTextContent("14 days remaining");
    expect(banner).toHaveAttribute("data-tone", "info");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });

  it("warns when the trial expires today", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          billingEnabled
          kind="trial"
          remainingCreditCents={4124}
          welcomeCreditExpiresAt={new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString()}
          spendingHref={billingHref}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveAttribute("data-tone", "warning");
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Expires today");
  });

  it("shows trial-empty copy", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled kind="trial-empty" spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Trial credit is empty");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });

  it("shows trial-expired copy", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled kind="trial-expired" spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Trial ended");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });
});
