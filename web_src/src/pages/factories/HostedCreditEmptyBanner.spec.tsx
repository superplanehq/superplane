import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { HostedCreditEmptyBanner } from "./HostedCreditEmptyBanner";

describe("HostedCreditEmptyBanner", () => {
  it("links to spending with a view action when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          level="empty"
          billingEnabled
          spendingHref="/org/workspaces/RF/settings/organization/spending"
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Hosted credit is empty");
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent(
      "Add hosted credit to start SuperPlane-hosted runs.",
    );
    expect(screen.getByRole("link", { name: "View spending" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/spending",
    );
  });

  it("links to spending with a view action when billing is off", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          level="empty"
          billingEnabled={false}
          spendingHref="/org/workspaces/RF/settings/organization/spending"
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "View spending" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/spending",
    );
  });

  it("shows View spending for a non-owner when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
          level="empty"
          billingEnabled
          canManageBilling={false}
          spendingHref="/org/workspaces/RF/settings/organization/spending"
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "View spending" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/spending",
    );
    expect(screen.queryByRole("link", { name: "Add hosted credit" })).not.toBeInTheDocument();
  });

  it("renders severe red styling for the empty level", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner level="empty" billingEnabled spendingHref="/spending" />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveClass("border-red-200");
  });

  it("renders warning amber styling and low-credit copy for the low level", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner level="low" billingEnabled spendingHref="/spending" />
      </MemoryRouter>,
    );

    const banner = screen.getByTestId("hosted-credit-empty-banner");
    expect(banner).toHaveClass("border-amber-200");
    expect(banner).toHaveTextContent("Hosted credit is running low");
  });

  it("renders a no-op go-to-billing button when no spending link is given", async () => {
    const user = userEvent.setup();
    const onGoToBilling = vi.fn();
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner level="empty" billingEnabled onGoToBilling={onGoToBilling} />
      </MemoryRouter>,
    );

    const button = screen.getByRole("button", { name: "Go to billing" });
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    await user.click(button);
    expect(onGoToBilling).toHaveBeenCalledTimes(1);
  });
});
