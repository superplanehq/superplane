import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { AccountContext, type AccountContextType } from "@/contexts/accountContextState";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { SpendingRedesignPage } from "./SpendingRedesignPage";
import { SPENDING_CATALOGS, SPENDING_CREDIT, SPENDING_LEDGER, SPENDING_REDESIGN_NOW } from "./spendingRedesignMocks";

const HOSTED_CREDIT_COPY =
  "SuperPlane-hosted model usage uses hosted credit. Bring your own key usage is estimated and is not billed.";

function accountValue(billingEnabled: boolean): AccountContextType {
  return {
    account: {
      id: "account-1",
      name: "Ada",
      email: "ada@example.com",
      avatar_url: "",
      installation_admin: false,
      has_password: true,
      billing_enabled: billingEnabled,
    },
    loading: false,
    setupRequired: false,
    refreshAccount: async () => undefined,
  };
}

function renderPage(billingEnabled: boolean) {
  return render(
    <AccountContext.Provider value={accountValue(billingEnabled)}>
      <ThemeProvider>
        <TooltipProvider>
          <SpendingRedesignPage
            catalogs={SPENDING_CATALOGS}
            credit={{ ...SPENDING_CREDIT, billingEnabled: false, hasBillingCustomer: false }}
            events={SPENDING_LEDGER}
            now={SPENDING_REDESIGN_NOW}
          />
        </TooltipProvider>
      </ThemeProvider>
    </AccountContext.Provider>,
  );
}

describe("SpendingRedesignPage billing", () => {
  it("hides hosted credit when billing is off and keeps usage reports", () => {
    renderPage(false);

    expect(screen.getByRole("heading", { name: "Model usage" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "VM usage" })).toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-hosted")).toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-vm")).toBeInTheDocument();
    expect(screen.queryByTestId("spending-kpi-credit")).not.toBeInTheDocument();
    expect(screen.queryByText("Remaining hosted credit")).not.toBeInTheDocument();
    expect(screen.getByText("Token usage and estimated spend by model.")).toBeInTheDocument();
    expect(screen.queryByText(HOSTED_CREDIT_COPY)).not.toBeInTheDocument();
  });

  it("keeps remaining hosted credit when Polar is off and product billing is on", () => {
    renderPage(true);

    expect(screen.getByTestId("spending-kpi-credit")).toHaveTextContent("$41.24");
    expect(screen.getByText("Remaining hosted credit")).toBeInTheDocument();
    expect(screen.getByText(HOSTED_CREDIT_COPY)).toBeInTheDocument();
  });
});
