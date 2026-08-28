import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HostedCreditSummary } from "./HostedCreditSummary";

describe("HostedCreditSummary", () => {
  it("shows credit packs as cards in ascending amount order", () => {
    render(
      <HostedCreditSummary
        remainingCreditCents="0"
        grantTotalCents="0"
        superplaneGrantCents="0"
        purchasedCreditCents="0"
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

  it("shows SuperPlane grant and purchased hosted credit separately", () => {
    render(
      <HostedCreditSummary
        remainingCreditCents="14630"
        grantTotalCents="15000"
        superplaneGrantCents="5000"
        purchasedCreditCents="10000"
        hostedBilledCents="370"
        billingEnabled
        cardClassName=""
        labelClassName=""
        valueClassName=""
      />,
    );

    expect(screen.getByText("SuperPlane grant")).toBeInTheDocument();
    expect(screen.getByText("Purchased hosted credit")).toBeInTheDocument();
    expect(screen.getByText("$50.00")).toBeInTheDocument();
    expect(screen.getByText("$100.00")).toBeInTheDocument();
    expect(screen.queryByText("Grant total")).not.toBeInTheDocument();
  });

  it("shows the Polar invoice empty state after a billing customer exists", () => {
    render(
      <HostedCreditSummary
        remainingCreditCents="2500"
        superplaneGrantCents="2500"
        purchasedCreditCents="0"
        hostedBilledCents="0"
        billingEnabled
        hasBillingCustomer
        canManageBilling
        cardClassName=""
        labelClassName=""
        valueClassName=""
      />,
    );

    expect(screen.getByText("No Polar invoices yet.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Manage invoices" })).toBeInTheDocument();
  });

  it("lists Polar invoices", () => {
    render(
      <HostedCreditSummary
        remainingCreditCents="14630"
        superplaneGrantCents="5000"
        purchasedCreditCents="10000"
        hostedBilledCents="370"
        billingEnabled
        hasBillingCustomer
        canManageBilling
        invoices={[
          {
            id: "ord_100",
            createdAt: "2026-08-27T12:00:00Z",
            amountCents: "10000",
            status: "paid",
            productName: "$100 pack",
          },
        ]}
        onManageInvoices={vi.fn()}
        cardClassName=""
        labelClassName=""
        valueClassName=""
      />,
    );

    expect(screen.getByText("$100 pack")).toBeInTheDocument();
    expect(screen.getByText("Paid")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Manage invoices" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open invoice" })).not.toBeInTheDocument();
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
