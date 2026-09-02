import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { HostedCreditEmptyBanner } from "./HostedCreditEmptyBanner";

describe("HostedCreditEmptyBanner", () => {
  it("links to spending with an add-credit action when billing is on", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner billingEnabled spendingHref="/org/workspaces/RF/settings/organization/spending" />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent("Hosted credit is empty");
    expect(screen.getByTestId("hosted-credit-empty-banner")).toHaveTextContent(
      "Add hosted credit to start SuperPlane-hosted runs.",
    );
    expect(screen.getByRole("link", { name: "Add hosted credit" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/spending",
    );
  });

  it("links to spending with a view action when billing is off", () => {
    render(
      <MemoryRouter>
        <HostedCreditEmptyBanner
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
});
