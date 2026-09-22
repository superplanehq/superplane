import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "bun:test";

import { HostedCreditHeaderKicker } from "./HostedCreditHeaderKicker";
import { welcomeCreditHeaderLabel } from "./lib/hostedCreditEmpty";

const billingHref = "/org/workspaces/RF/settings/organization/billing";

describe("HostedCreditHeaderKicker", () => {
  it("shows remaining trial days and a Subscribe action", () => {
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
    expect(kicker).toHaveAttribute("data-kind", "trial");
    expect(kicker).toHaveTextContent("Trial");
    expect(kicker).toHaveTextContent(welcomeCreditHeaderLabel(expiresAt));
    expect(kicker).toHaveTextContent("$41.24");
    expect(kicker).toHaveTextContent("Subscribe");
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute("href", billingHref);
  });

  it("keeps the violet palette for the trial chip", () => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker spendingHref={billingHref} remainingCreditCents={4124} />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker.className).toContain("bg-violet-100");
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

  it("shows a no-plan chip with a Subscribe action", () => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker kind="lapsed" spendingHref={billingHref} remainingCreditCents={5000} />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker).toHaveAttribute("data-kind", "lapsed");
    expect(kicker).toHaveTextContent("No plan");
    expect(kicker).toHaveTextContent("Subscribe");
    expect(kicker).not.toHaveTextContent("$50.00");
    expect(screen.getByRole("link", { name: "Subscribe" })).toHaveAttribute("href", billingHref);
  });

  it("shows a trial-ended chip with a Subscribe action", () => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker kind="trial-expired" spendingHref={billingHref} />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker).toHaveAttribute("data-kind", "trial-expired");
    expect(kicker).toHaveTextContent("Trial ended");
    expect(kicker).toHaveTextContent("Subscribe");
  });

  it("shows the remaining balance on a low-credit chip with an Add credits action", () => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker kind="low" remainingCreditCents={1500} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker).toHaveAttribute("data-kind", "low");
    expect(kicker).toHaveTextContent("Credit low");
    expect(kicker).toHaveTextContent("$15.00");
    expect(kicker).toHaveTextContent("Add credits");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });

  it("shows only the label and action on an empty-credit chip", () => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker kind="empty" remainingCreditCents={0} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker).toHaveAttribute("data-kind", "empty");
    expect(kicker).toHaveTextContent("No credit");
    expect(kicker).toHaveTextContent("Add credits");
    expect(kicker).not.toHaveTextContent("$0.00");
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", billingHref);
  });

  it.each(["low", "empty"] as const)("uses the amber palette for the %s chip", (kind) => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker
          kind={kind}
          remainingCreditCents={kind === "low" ? 1500 : 0}
          spendingHref={billingHref}
        />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker.className).toContain("bg-amber-100");
  });

  it.each(["trial-expired", "lapsed"] as const)("uses the amber palette for the %s chip", (kind) => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker kind={kind} spendingHref={billingHref} />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker.className).toContain("bg-amber-100");
  });

  it("hides the action pill for low and empty credit when credit cannot be added", () => {
    render(
      <MemoryRouter>
        <HostedCreditHeaderKicker
          kind="low"
          remainingCreditCents={1500}
          spendingHref={billingHref}
          canAddCredit={false}
        />
      </MemoryRouter>,
    );

    const kicker = screen.getByTestId("hosted-credit-header-kicker");
    expect(kicker).toHaveTextContent("Credit low");
    expect(kicker).not.toHaveTextContent("Add credits");
    expect(screen.getByRole("link")).toHaveAttribute("href", billingHref);
  });
});
