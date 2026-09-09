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
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { hostedCreditBillingBalanceCopy } from "../../lib/hostedCreditEmpty";
import { creditGrantDetails, creditGrantSourceLabel, formatCreditGrantAmount } from "../../lib/hostedCreditGrants";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { useOrganizationBillingPageModel } from "./useOrganizationBillingPageModel";

export function OrganizationSettingsBillingPage() {
  const { organizationId = "", factoryKey = "" } = useParams<{ organizationId: string; factoryKey: string }>();
  const model = useOrganizationBillingPageModel(organizationId);
  const packs = sortHostedCreditPacks(model.billing.products);
  const showStackedPacks = model.hasBillingCustomer && model.canManageBilling && packs.length > 0;

  usePageTitle(["Billing", model.organizationName]);
  useReportPageReady(!model.isLoading, { failed: Boolean(model.error) });

  return (
    <FactorySettingsPageFrame
      title="Billing"
      subtitle="Hosted credit pays SuperPlane-hosted models for this organization."
      actions={
        showStackedPacks ? undefined : (
          <AddHostedCreditAction
            canManageBilling={model.canManageBilling}
            checkoutPending={model.billing.checkoutPending}
            products={packs}
            onAddCredit={model.billing.startCheckout}
          />
        )
      }
    >
      <BillingPageBody
        billingContactMessage={model.billingContactMessage}
        billingEnabled={model.billingEnabled}
        canManageBilling={model.canManageBilling}
        checkoutPending={model.billing.checkoutPending}
        creditRefreshStatus={model.creditRefreshStatus}
        error={model.error}
        factoryKey={factoryKey}
        grants={model.grants}
        hasBillingCustomer={model.hasBillingCustomer}
        invoices={model.invoices}
        isLoading={model.isLoading}
        organizationId={organizationId}
        packs={packs}
        portalPending={model.billing.portalPending}
        purchased={model.purchased}
        remaining={model.remaining}
        showStackedPacks={showStackedPacks}
        welcomeCreditExpiresAt={model.welcomeCreditExpiresAt}
        onAddCredit={model.billing.startCheckout}
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

  const label = checkoutPending ? "Opening checkout..." : "Add hosted credit";

  if (products.length === 0) {
    return (
      <Button type="button" disabled>
        {label}
      </Button>
    );
  }

  if (products.length === 1) {
    const productId = products[0].id ?? "";
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
        {products.map((product) => {
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
  billingContactMessage,
  billingEnabled,
  canManageBilling,
  checkoutPending,
  creditRefreshStatus,
  error,
  factoryKey,
  grants,
  hasBillingCustomer,
  invoices,
  isLoading,
  organizationId,
  packs,
  portalPending,
  purchased,
  remaining,
  showStackedPacks,
  welcomeCreditExpiresAt,
  onAddCredit,
  onManageInvoices,
}: {
  billingContactMessage?: string;
  billingEnabled: boolean;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  error: unknown;
  factoryKey: string;
  grants: OrganizationsOrganizationCreditGrant[];
  hasBillingCustomer: boolean;
  invoices: OrganizationsHostedCreditInvoice[];
  isLoading: boolean;
  organizationId: string;
  packs: OrganizationsHostedCreditProduct[];
  portalPending: boolean;
  purchased: number;
  remaining: number;
  showStackedPacks: boolean;
  welcomeCreditExpiresAt?: string;
  onAddCredit: (productId: string) => void | Promise<void>;
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

  const spendingHref = factorySettingsSectionPath(organizationId, factoryKey, "organization", "spending");
  const showInvoices = billingEnabled && canManageBilling && hasBillingCustomer;

  return (
    <>
      <HostedCreditRemainingCard
        billingContactMessage={billingContactMessage}
        billingEnabled={billingEnabled}
        canManageBilling={canManageBilling}
        checkoutPending={checkoutPending}
        creditRefreshStatus={creditRefreshStatus}
        hasBillingCustomer={hasBillingCustomer}
        packs={packs}
        purchased={purchased}
        remaining={remaining}
        showStackedPacks={showStackedPacks}
        spendingHref={spendingHref}
        welcomeCreditExpiresAt={welcomeCreditExpiresAt}
        onAddCredit={onAddCredit}
      />
      {showInvoices ? (
        <PolarInvoicesCard invoices={invoices} portalPending={portalPending} onManageInvoices={onManageInvoices} />
      ) : null}
      <CreditHistoryCard grants={grants} />
    </>
  );
}

function HostedCreditRemainingCard({
  billingContactMessage,
  billingEnabled,
  canManageBilling,
  checkoutPending,
  creditRefreshStatus,
  hasBillingCustomer,
  packs,
  purchased,
  remaining,
  showStackedPacks,
  spendingHref,
  welcomeCreditExpiresAt,
  onAddCredit,
}: {
  billingContactMessage?: string;
  billingEnabled: boolean;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  hasBillingCustomer: boolean;
  packs: OrganizationsHostedCreditProduct[];
  purchased: number;
  remaining: number;
  showStackedPacks: boolean;
  spendingHref: string;
  welcomeCreditExpiresAt?: string;
  onAddCredit: (productId: string) => void | Promise<void>;
}) {
  const copy = hostedCreditBillingBalanceCopy({
    remainingCents: remaining,
    purchasedCents: purchased,
    hasBillingCustomer,
    billingEnabled,
    welcomeCreditExpiresAt,
  });
  const creditRefreshMessage = hostedCreditRefreshMessage(creditRefreshStatus);

  return (
    <FactorySettingsCard title="Hosted credit" data-testid="billing-credit-balance">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="workspace-section-label">Remaining hosted credit</p>
            {copy.badge ? (
              <Badge variant="outline" className="text-muted-foreground">
                {copy.badge}
              </Badge>
            ) : null}
          </div>
          <p className="workspace-page-title mt-1" data-testid="billing-remaining-credit">
            {formatUsdCents(remaining)}
          </p>
          {creditRefreshMessage ? (
            <p className={`mt-3 text-sm ${creditRefreshClassName(creditRefreshStatus)}`}>{creditRefreshMessage}</p>
          ) : null}
          {copy.description ? (
            <p
              className={cn(
                "mt-3 text-sm",
                remaining <= 0 ? "text-amber-700 dark:text-amber-400" : "text-muted-foreground",
              )}
            >
              {copy.description}
            </p>
          ) : null}
          {!canManageBilling && billingContactMessage ? (
            <p className="mt-3 text-sm text-muted-foreground">{billingContactMessage}</p>
          ) : null}
          <p className="mt-3 text-sm text-muted-foreground">
            <Link className="underline underline-offset-2" to={spendingHref}>
              View spending
            </Link>
          </p>
        </div>
        {showStackedPacks ? (
          <HostedCreditPackButtons checkoutPending={checkoutPending} packs={packs} onAddCredit={onAddCredit} />
        ) : null}
      </div>
    </FactorySettingsCard>
  );
}

function HostedCreditPackButtons({
  checkoutPending,
  packs,
  onAddCredit,
}: {
  checkoutPending: boolean;
  packs: OrganizationsHostedCreditProduct[];
  onAddCredit: (productId: string) => void | Promise<void>;
}) {
  return (
    <div className="flex w-full flex-col gap-2 sm:w-44" data-testid="billing-credit-packs">
      {packs.map((product) => {
        const productId = product.id ?? "";
        const amount = parseWorkOrderMetric(product.amountCents);
        return (
          <Button
            key={productId || amount}
            type="button"
            variant="outline"
            disabled={!productId || checkoutPending}
            onClick={() => productId && void onAddCredit(productId)}
          >
            {checkoutPending ? "Opening checkout..." : `Add ${formatUsdCents(amount)}`}
          </Button>
        );
      })}
    </div>
  );
}

function PolarInvoicesCard({
  invoices,
  portalPending,
  onManageInvoices,
}: {
  invoices: OrganizationsHostedCreditInvoice[];
  portalPending: boolean;
  onManageInvoices: () => void | Promise<void>;
}) {
  return (
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
  );
}

function CreditHistoryCard({ grants }: { grants: OrganizationsOrganizationCreditGrant[] }) {
  return (
    <FactorySettingsCard title="Credit history" data-testid="billing-credit-history">
      {grants.length === 0 ? (
        <p className="text-sm text-muted-foreground">No credit grants yet.</p>
      ) : (
        <CreditGrantTable grants={grants} />
      )}
    </FactorySettingsCard>
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
