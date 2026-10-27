import type { OrganizationsHostedCreditInvoice, OrganizationsOrganizationCreditGrant } from "@/api-client";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { BillingInvoicesCard } from "./BillingInvoicesCard";
import { BillingCreditHistoryCard } from "./BillingCreditHistoryCard";

export function BillingHistory({
  showInvoices,
  invoices,
  grants,
  portalPending,
  onManageInvoices,
}: {
  showInvoices: boolean;
  invoices: OrganizationsHostedCreditInvoice[];
  grants: OrganizationsOrganizationCreditGrant[];
  portalPending: boolean;
  onManageInvoices: () => void | Promise<void>;
}) {
  const invoiceList = (
    <BillingInvoicesCard invoices={invoices} portalPending={portalPending} onManageInvoices={onManageInvoices} />
  );
  const creditHistory = <BillingCreditHistoryCard grants={grants} />;
  if (!showInvoices) return creditHistory;
  return (
    <Tabs defaultValue="invoices">
      <TabsList aria-label="Billing history" className="mb-2">
        <TabsTrigger value="invoices">Invoices</TabsTrigger>
        <TabsTrigger value="credits">Credit history</TabsTrigger>
      </TabsList>
      <TabsContent value="invoices">{invoiceList}</TabsContent>
      <TabsContent value="credits">{creditHistory}</TabsContent>
    </Tabs>
  );
}
