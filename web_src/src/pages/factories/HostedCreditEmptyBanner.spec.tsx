import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { HostedCreditEmptyBanner, HostedCreditHeaderKicker } from "./HostedCreditEmptyBanner";
import { HOSTED_CREDIT_RUNS_STOP_HINT, welcomeCreditHeaderLabel } from "./lib/hostedCreditEmpty";

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

  it("hides the action when billing is off", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled={false} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.queryByRole("link", { name: "Add credits" })).not.toBeInTheDocument();
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent(
      "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
    );
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

  it("shows remaining trial credit and a Subscribe action", () => {
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
    expect(banner).not.toHaveTextContent(HOSTED_CREDIT_RUNS_STOP_HINT);
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute("href", billingHref);
    expect(screen.getByRole("link", { name: "See pricing" })).toHaveAttribute(
      "href",
      "https://superplane.com/pricing/",
    );
  });

  it("warns that tasks stop when remaining trial credit is low", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          billingEnabled
          kind="trial"
          remainingCreditCents={432}
          welcomeCreditExpiresAt={new Date(Date.now() + 14 * 24 * 60 * 60 * 1000).toISOString()}
          spendingHref={billingHref}
        />
      </MemoryRouter>,
    );

    const banner = screen.getByTestId("hosted-credit-empty-banner");
    expect(banner).toHaveTextContent("Trial");
    expect(banner).toHaveTextContent("$4.32 remaining");
    expect(banner).toHaveTextContent(HOSTED_CREDIT_RUNS_STOP_HINT);
    expect(banner).toHaveAttribute("data-tone", "warning");
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

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Trial credit is used up");
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute("href", billingHref);
  });

  it("shows trial-expired copy", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled kind="trial-expired" spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Trial ended");
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute("href", billingHref);
  });

  it("shows remaining purchased credit when the balance is low", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled kind="low" remainingCreditCents={1500} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    const banner = screen.getByTestId("hosted-credit-empty-banner");
    expect(banner).toHaveTextContent("Hosted credit is low");
    expect(banner).toHaveTextContent("$15.00 remaining");
    expect(banner).toHaveTextContent(HOSTED_CREDIT_RUNS_STOP_HINT);
    expect(banner).toHaveAttribute("data-tone", "warning");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });
});

describe("HostedCreditHeaderKicker", () => {
  it("shows remaining trial days and an Add credits action", () => {
    const expiresAt = new Date(Date.now() + 13 * 24 * 60 * 60 * 1000);
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker
          spendingHref={billingHref}
          welcomeCreditExpiresAt={expiresAt.toISOString()}
          remainingCreditCents={4124}
        />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker).toHaveTextContent("Trial");
    expect(kicker).toHaveTextContent(welcomeCreditHeaderLabel(expiresAt));
    expect(kicker).toHaveTextContent("$41.24");
    expect(kicker).toHaveTextContent("Add credits");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });

  it("falls back to a 14-day trial label when expiry is missing", () => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker spendingHref={billingHref} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-header-kicker")).toHaveTextContent("Trial");
    expect(screen.getByTestId("hosted-credit-header-kicker")).toHaveTextContent("14 days");
  });
});
