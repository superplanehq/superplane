import { ExternalLink } from "lucide-react";
import { Link, useParams } from "react-router";

import type {
  OrganizationsHostedCreditInvoice,
  OrganizationsHostedCreditProduct,
  OrganizationsOrganizationCreditGrant,
} from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedCreditRefreshMessage, type HostedCreditRefreshStatus } from "@/lib/hostedCredit";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import {
  creditGrantDetails,
  creditGrantSourceLabel,
  formatCreditGrantAmount,
  hostedCreditBalanceWarning,
  welcomeCreditUnusedExpiryNote,
} from "../../lib/hostedCreditGrants";
import { isWelcomeCreditExpired } from "../../lib/hostedCreditEmpty";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { useOrganizationBillingPageModel } from "./useOrganizationBillingPageModel";

export function OrganizationSettingsBillingPage() {
  const { organizationId = "", factoryKey = "" } = useParams<{ organizationId: string; factoryKey: string }>();
  const model = useOrganizationBillingPageModel(organizationId);

  usePageTitle(["Billing", model.organizationName]);
  useReportPageReady(!model.isLoading, { failed: Boolean(model.error) });

  return (
    <FactorySettingsPageFrame
      title="Billing"
      subtitle="Hosted credit pays SuperPlane-hosted models for this organization."
      actions={
        <AddHostedCreditAction
          canManageBilling={model.canManageBilling}
          checkoutPending={model.billing.checkoutPending}
          products={model.billing.products}
          onAddCredit={model.billing.startCheckout}
        />
      }
    >
      <BillingPageBody
        billed={model.billed}
        billingContactMessage={model.billingContactMessage}
        billingEnabled={model.billingEnabled}
        canManageBilling={model.canManageBilling}
        creditRefreshStatus={model.creditRefreshStatus}
        error={model.error}
        factoryKey={factoryKey}
        grants={model.grants}
        hasBillingCustomer={model.hasBillingCustomer}
        invoices={model.invoices}
        isLoading={model.isLoading}
        organizationId={organizationId}
        portalPending={model.billing.portalPending}
        purchased={model.purchased}
        remaining={model.remaining}
        remainingCreditWarning={model.remainingCreditWarning}
        superplaneGrant={model.superplaneGrant}
        welcomeCreditExpiresAt={model.welcomeCreditExpiresAt}
        onManageInvoices={model.billing.openInvoices}
      />
    </FactorySettingsPageFrame>
  );
}

function AddHostedCreditAction({
  canManageBilling,
  checkoutPending,
  products,
  onAddCredit,
}: {
  canManageBilling: boolean;
  checkoutPending: boolean;
  products: OrganizationsHostedCreditProduct[];
  onAddCredit: (productId: string) => void | Promise<void>;
}) {
  if (!canManageBilling) {
    return null;
  }

  const packs = sortHostedCreditPacks(products);
  const label = checkoutPending ? "Opening checkout..." : "Add hosted credit";

  if (packs.length === 0) {
    return (
      <Button type="button" disabled>
        {label}
      </Button>
    );
  }

  if (packs.length === 1) {
    const productId = packs[0].id ?? "";
    return (
      <Button
        type="button"
        disabled={!productId || checkoutPending}
        onClick={() => productId && void onAddCredit(productId)}
      >
        {label}
      </Button>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" disabled={checkoutPending}>
          {label}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {packs.map((product) => {
          const productId = product.id ?? "";
          const amount = parseWorkOrderMetric(product.amountCents);
          return (
            <DropdownMenuItem
              key={productId || amount}
              disabled={!productId || checkoutPending}
              onClick={() => productId && void onAddCredit(productId)}
            >
              {formatUsdCents(amount)}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function BillingPageBody({
  billed,
  billingContactMessage,
  billingEnabled,
  canManageBilling,
  creditRefreshStatus,
  error,
  factoryKey,
  grants,
  hasBillingCustomer,
  invoices,
  isLoading,
  organizationId,
  portalPending,
  purchased,
  remaining,
  remainingCreditWarning,
  superplaneGrant,
  welcomeCreditExpiresAt,
  onManageInvoices,
}: {
  billed: number;
  billingContactMessage?: string;
  billingEnabled: boolean;
  canManageBilling: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  error: unknown;
  factoryKey: string;
  grants: OrganizationsOrganizationCreditGrant[];
  hasBillingCustomer: boolean;
  invoices: OrganizationsHostedCreditInvoice[];
  isLoading: boolean;
  organizationId: string;
  portalPending: boolean;
  purchased: number;
  remaining: number;
  remainingCreditWarning: boolean;
  superplaneGrant: number;
  welcomeCreditExpiresAt?: string;
  onManageInvoices: () => void | Promise<void>;
}) {
  if (isLoading) {
    return (
      <FactorySettingsCard>
        <p className="text-sm text-muted-foreground">Loading billing...</p>
      </FactorySettingsCard>
    );
  }

  if (error) {
    return (
      <FactorySettingsCard>
        <p className="text-sm text-destructive">{getApiErrorMessage(error, "Unable to load billing.")}</p>
      </FactorySettingsCard>
    );
  }

  const warning = hostedCreditBalanceWarning(
    remaining,
    remainingCreditWarning,
    isWelcomeCreditExpired(welcomeCreditExpiresAt) && purchased === 0,
  );
  const expiryNote = welcomeCreditUnusedExpiryNote({
    remainingCents: remaining,
    purchasedCents: purchased,
    welcomeCreditExpiresAt,
  });
  const creditRefreshMessage = hostedCreditRefreshMessage(creditRefreshStatus);
  const spendingHref = factorySettingsSectionPath(organizationId, factoryKey, "organization", "spending");
  const showInvoices = billingEnabled && canManageBilling && hasBillingCustomer;

  return (
    <>
      <FactorySettingsCard title="Hosted credit" data-testid="billing-credit-balance">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <CreditMetric label="Remaining hosted credit" value={remaining} hint="Organization wallet" />
          <CreditMetric label="SuperPlane grant" value={superplaneGrant} hint="Welcome and support grants" />
          <CreditMetric label="Purchased hosted credit" value={purchased} hint="Polar credit packs" />
          <CreditMetric label="Hosted billed spend" value={billed} hint="SuperPlane-hosted models" />
        </div>
        {creditRefreshMessage ? (
          <p className={`mt-3 text-sm ${creditRefreshClassName(creditRefreshStatus)}`}>{creditRefreshMessage}</p>
        ) : null}
        {warning ? <p className="mt-3 text-sm text-amber-700 dark:text-amber-400">{warning}</p> : null}
        {expiryNote ? <p className="mt-3 text-sm text-muted-foreground">{expiryNote}</p> : null}
        {!canManageBilling && billingContactMessage ? (
          <p className="mt-3 text-sm text-muted-foreground">{billingContactMessage}</p>
        ) : null}
        <p className="mt-3 text-sm text-muted-foreground">
          <Link className="underline underline-offset-2" to={spendingHref}>
            View spending
          </Link>
        </p>
      </FactorySettingsCard>

      {showInvoices ? (
        <FactorySettingsCard
          title="Polar invoices"
          data-testid="billing-polar-invoices"
          action={
            <Button type="button" variant="ghost" disabled={portalPending} onClick={() => void onManageInvoices()}>
              {portalPending ? "Opening invoices..." : "Manage invoices"}
              <ExternalLink className="size-3.5" aria-hidden />
            </Button>
          }
        >
          {invoices.length === 0 ? (
            <p className="text-sm text-muted-foreground">No Polar invoices yet.</p>
          ) : (
            <InvoiceTable invoices={invoices} />
          )}
        </FactorySettingsCard>
      ) : null}

      <FactorySettingsCard title="Credit history" data-testid="billing-credit-history">
        {grants.length === 0 ? (
          <p className="text-sm text-muted-foreground">No credit grants yet.</p>
        ) : (
          <CreditGrantTable grants={grants} />
        )}
      </FactorySettingsCard>
    </>
  );
}

function CreditMetric({ label, value, hint }: { label: string; value: number; hint: string }) {
  return (
    <div>
      <p className="workspace-section-label">{label}</p>
      <p className="workspace-page-title mt-1">{formatUsdCents(value)}</p>
      <p className="mt-1 text-[12px] text-muted-foreground">{hint}</p>
    </div>
  );
}

function CreditGrantTable({ grants }: { grants: OrganizationsOrganizationCreditGrant[] }) {
  return (
    <table className="mt-1 w-full text-left text-[13px]">
      <thead>
        <tr className="border-b border-border text-muted-foreground">
          <th className="py-2 pr-3 font-medium">Date</th>
          <th className="py-2 pr-3 font-medium">Source</th>
          <th className="py-2 pr-3 font-medium">Amount</th>
          <th className="py-2 font-medium">Details</th>
        </tr>
      </thead>
      <tbody>
        {grants.map((grant) => (
          <tr key={grant.id ?? `${grant.kind}-${grant.createdAt}`} className="border-b border-border last:border-0">
            <td className="py-2 pr-3 whitespace-nowrap">{formatGrantDate(grant.createdAt)}</td>
            <td className="py-2 pr-3">
              <Badge variant="secondary">{creditGrantSourceLabel(grant.kind)}</Badge>
            </td>
            <td className="py-2 pr-3 whitespace-nowrap">
              {formatCreditGrantAmount(parseWorkOrderMetric(grant.amountCents))}
            </td>
            <td className="py-2 text-muted-foreground">{creditGrantDetails(grant) || "—"}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function InvoiceTable({ invoices }: { invoices: OrganizationsHostedCreditInvoice[] }) {
  return (
    <table className="mt-1 w-full text-left text-[13px]">
      <thead>
        <tr className="border-b border-border text-muted-foreground">
          <th className="py-2 font-medium">Date</th>
          <th className="py-2 font-medium">Item</th>
          <th className="py-2 font-medium">Amount</th>
          <th className="py-2 font-medium">Status</th>
        </tr>
      </thead>
      <tbody>
        {invoices.map((invoice) => (
          <tr key={invoice.id} className="border-b border-border last:border-0">
            <td className="py-2">{formatGrantDate(invoice.createdAt)}</td>
            <td className="py-2">{invoice.productName || "Hosted credit"}</td>
            <td className="py-2">{formatUsdCents(parseWorkOrderMetric(invoice.amountCents))}</td>
            <td className="py-2">{invoiceStatusLabel(invoice.status)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function sortHostedCreditPacks(products: OrganizationsHostedCreditProduct[]) {
  return products
    .slice()
    .sort((left, right) => parseWorkOrderMetric(left.amountCents) - parseWorkOrderMetric(right.amountCents));
}

function creditRefreshClassName(status: HostedCreditRefreshStatus) {
  if (status === "added") {
    return "text-emerald-700 dark:text-emerald-400";
  }
  return "text-muted-foreground";
}

function formatGrantDate(value: string | undefined) {
  if (!value) {
    return "—";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleDateString();
}

function invoiceStatusLabel(status: string | undefined) {
  switch (status) {
    case "paid":
      return "Paid";
    case "refunded":
      return "Refunded";
    case "partially_refunded":
      return "Partially refunded";
    default:
      return status || "Unknown";
  }
}
