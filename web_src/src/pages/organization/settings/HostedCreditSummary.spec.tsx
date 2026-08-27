import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HostedCreditSummary } from "./HostedCreditSummary";

describe("HostedCreditSummary", () => {
  it("shows credit packs as cards in ascending amount order", () => {
    render(
      <HostedCreditSummary
        remainingCreditCents="0"
        grantTotalCents="0"
        hostedBilledCents="0"
        remainingCreditWarning
        billingEnabled
        canManageBilling
        products={[
          { id: "prod-500", amountCents: "50000" },
          { id: "prod-25", amountCents: "2500" },
          { id: "prod-100", amountCents: "10000" },
        ]}
        onAddCredit={vi.fn()}
        cardClassName=""
        labelClassName=""
        valueClassName=""
      />,
    );

    const amounts = screen.getAllByText(/^\$\d+\.\d{2}$/).map((node) => node.textContent);
    expect(amounts.filter((value) => value === "$25.00" || value === "$100.00" || value === "$500.00")).toEqual([
      "$25.00",
      "$100.00",
      "$500.00",
    ]);
    expect(screen.getAllByRole("button", { name: "Add hosted credit" })).toHaveLength(3);
  });

  it("shows the credit refresh message when checkout returns", () => {
    render(
      <HostedCreditSummary
        remainingCreditCents="2500"
        grantTotalCents="2500"
        hostedBilledCents="0"
        billingEnabled
        creditRefreshStatus="refreshing"
        cardClassName=""
        labelClassName=""
        valueClassName=""
      />,
    );

    expect(screen.getByText("Refreshing hosted credit totals.")).toBeInTheDocument();
  });
});
