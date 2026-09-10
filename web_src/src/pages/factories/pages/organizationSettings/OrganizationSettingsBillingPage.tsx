import { ChevronDown, ExternalLink } from "lucide-react";
import { useParams } from "react-router";

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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";

import { hostedCreditBillingBalanceCopy } from "../../lib/hostedCreditEmpty";
import { creditGrantDetails, creditGrantSourceLabel, formatCreditGrantAmount } from "../../lib/hostedCreditGrants";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { useOrganizationBillingPageModel } from "./useOrganizationBillingPageModel";

const BUY_MORE_PACK_CENTS = [5_000, 10_000, 50_000] as const;
const HOSTED_CREDIT_EXPLANATION =
  "Hosted credit pays SuperPlane-hosted machines and managed models for this organization.";

export function OrganizationSettingsBillingPage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  const model = useOrganizationBillingPageModel(organizationId);
  const packs = model.billing.products;

  usePageTitle(["Billing", model.organizationName]);
  useReportPageReady(!model.isLoading, { failed: Boolean(model.error) });

  return (
    <FactorySettingsPageFrame title="Billing">
      <BillingPageBody
        billingContactMessage={model.billingContactMessage}
        billingEnabled={model.billingEnabled}
        businessCheckoutPending={model.billing.businessCheckoutPending}
        canManageBilling={model.canManageBilling}
        checkoutPending={model.billing.checkoutPending}
        creditPurchaseAllowed={model.creditPurchaseAllowed}
        creditRefreshStatus={model.creditRefreshStatus}
        error={model.error}
        grants={model.grants}
        hasBillingCustomer={model.hasBillingCustomer}
        invoices={model.invoices}
        isLoading={model.isLoading}
        packs={packs}
        plan={model.plan}
        portalPending={model.billing.portalPending}
        purchased={model.purchased}
        remaining={model.remaining}
        subscriptionCheckoutEnabled={model.subscriptionCheckoutEnabled}
        trialEndsAt={model.trialEndsAt}
        welcomeCreditExpiresAt={model.welcomeCreditExpiresAt}
        onAddCredit={model.billing.startCheckout}
        onSubscribe={model.billing.startBusinessCheckout}
        onManageInvoices={model.billing.openInvoices}
      />
    </FactorySettingsPageFrame>
  );
}

function BillingPageBody({
  billingContactMessage,
  billingEnabled,
  businessCheckoutPending,
  canManageBilling,
  checkoutPending,
  creditPurchaseAllowed,
  creditRefreshStatus,
  error,
  grants,
  hasBillingCustomer,
  invoices,
  isLoading,
  packs,
  plan,
  portalPending,
  purchased,
  remaining,
  subscriptionCheckoutEnabled,
  trialEndsAt,
  welcomeCreditExpiresAt,
  onAddCredit,
  onSubscribe,
  onManageInvoices,
}: {
  billingContactMessage?: string;
  billingEnabled: boolean;
  businessCheckoutPending: boolean;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditPurchaseAllowed: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  error: unknown;
  grants: OrganizationsOrganizationCreditGrant[];
  hasBillingCustomer: boolean;
  invoices: OrganizationsHostedCreditInvoice[];
  isLoading: boolean;
  packs: OrganizationsHostedCreditProduct[];
  plan?: string;
  portalPending: boolean;
  purchased: number;
  remaining: number;
  subscriptionCheckoutEnabled: boolean;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  onAddCredit: (productId: string) => void | Promise<void>;
  onSubscribe: () => void | Promise<void>;
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

  const showInvoices = billingEnabled && canManageBilling && hasBillingCustomer;

  return (
    <>
      <HostedCreditRemainingCard
        billingContactMessage={billingContactMessage}
        billingEnabled={billingEnabled}
        businessCheckoutPending={businessCheckoutPending}
        canManageBilling={canManageBilling}
        checkoutPending={checkoutPending}
        creditPurchaseAllowed={creditPurchaseAllowed}
        creditRefreshStatus={creditRefreshStatus}
        hasBillingCustomer={hasBillingCustomer}
        packs={packs}
        plan={plan}
        purchased={purchased}
        remaining={remaining}
        subscriptionCheckoutEnabled={subscriptionCheckoutEnabled}
        trialEndsAt={trialEndsAt}
        welcomeCreditExpiresAt={welcomeCreditExpiresAt}
        onAddCredit={onAddCredit}
        onSubscribe={onSubscribe}
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
  businessCheckoutPending,
  canManageBilling,
  checkoutPending,
  creditPurchaseAllowed,
  creditRefreshStatus,
  hasBillingCustomer,
  packs,
  plan,
  purchased,
  remaining,
  subscriptionCheckoutEnabled,
  trialEndsAt,
  welcomeCreditExpiresAt,
  onAddCredit,
  onSubscribe,
}: {
  billingContactMessage?: string;
  billingEnabled: boolean;
  businessCheckoutPending: boolean;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditPurchaseAllowed: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  hasBillingCustomer: boolean;
  packs: OrganizationsHostedCreditProduct[];
  plan?: string;
  purchased: number;
  remaining: number;
  subscriptionCheckoutEnabled: boolean;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  onAddCredit: (productId: string) => void | Promise<void>;
  onSubscribe: () => void | Promise<void>;
}) {
  const copy = hostedCreditBillingBalanceCopy({
    remainingCents: remaining,
    purchasedCents: purchased,
    hasBillingCustomer,
    billingEnabled,
    welcomeCreditExpiresAt,
    plan,
    trialEndsAt,
    creditPurchaseAllowed,
  });
  const creditRefreshMessage = hostedCreditRefreshMessage(creditRefreshStatus);
  const showSubscribe = canManageBilling && subscriptionCheckoutEnabled && !creditPurchaseAllowed;
  const showBuyMore = canManageBilling && creditPurchaseAllowed;

  return (
    <FactorySettingsCard
      title="Hosted credit"
      data-testid="billing-credit-balance"
      action={
        showBuyMore ? (
          <HostedCreditBuyMoreMenu checkoutPending={checkoutPending} packs={packs} onAddCredit={onAddCredit} />
        ) : showSubscribe ? (
          <SubscribeActions pending={businessCheckoutPending} onSubscribe={onSubscribe} />
        ) : undefined
      }
    >
      <p className="text-sm text-muted-foreground">{HOSTED_CREDIT_EXPLANATION}</p>
      <div className="mt-4 min-w-0">
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
      </div>
    </FactorySettingsCard>
  );
}

function SubscribeActions({ pending, onSubscribe }: { pending: boolean; onSubscribe: () => void | Promise<void> }) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button asChild variant="ghost" size="sm">
        <a href={SUPERPLANE_PRICING_URL} target="_blank" rel="noreferrer">
          See pricing
        </a>
      </Button>
      <Button type="button" disabled={pending} data-testid="billing-subscribe" onClick={() => void onSubscribe()}>
        {pending ? "Opening checkout..." : "Subscribe"}
      </Button>
    </div>
  );
}

function HostedCreditBuyMoreMenu({
  checkoutPending,
  packs,
  onAddCredit,
}: {
  checkoutPending: boolean;
  packs: OrganizationsHostedCreditProduct[];
  onAddCredit: (productId: string) => void | Promise<void>;
}) {
  const hasPurchasablePack = BUY_MORE_PACK_CENTS.some((cents) => findPackForCents(packs, cents));

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" disabled={checkoutPending || !hasPurchasablePack} data-testid="billing-buy-more">
          {checkoutPending ? "Opening checkout..." : "Buy more"}
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {BUY_MORE_PACK_CENTS.map((cents) => {
          const product = findPackForCents(packs, cents);
          const productId = product?.id ?? "";
          return (
            <DropdownMenuItem
              key={cents}
              disabled={!productId || checkoutPending}
              onClick={() => productId && void onAddCredit(productId)}
            >
              {formatUsdPackLabel(cents)}
            </DropdownMenuItem>
          );
        })}
        <DropdownMenuSeparator />
        <DropdownMenuItem disabled title="Custom amounts are not available yet.">
          Custom
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
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
      title="Invoices"
      data-testid="billing-polar-invoices"
      action={
        <Button type="button" variant="ghost" disabled={portalPending} onClick={() => void onManageInvoices()}>
          {portalPending ? "Opening invoices..." : "Manage invoices"}
          <ExternalLink className="size-3.5" aria-hidden />
        </Button>
      }
    >
      {invoices.length === 0 ? (
        <p className="text-sm text-muted-foreground">No invoices yet.</p>
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

function findPackForCents(packs: OrganizationsHostedCreditProduct[], cents: number) {
  return packs.find((product) => Boolean(product.id) && parseWorkOrderMetric(product.amountCents) === cents);
}

function formatUsdPackLabel(cents: number): string {
  const dollars = cents / 100;
  if (Number.isInteger(dollars)) {
    return `$${dollars}`;
  }
  return formatUsdCents(cents);
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
