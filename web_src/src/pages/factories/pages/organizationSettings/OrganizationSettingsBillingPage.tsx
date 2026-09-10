import { Check, ChevronDown, ExternalLink, Receipt } from "lucide-react";
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

import {
  BILLING_SPEND_ORDER_COPY,
  billingCreditBucketsView,
  billingCreditRemainingShares,
  type BillingCreditBarShare,
  type BillingCreditBucketKey,
  type BillingCreditBucketView,
} from "../../lib/billingCreditBuckets";
import { hostedCreditBillingBalanceCopy } from "../../lib/hostedCreditEmpty";
import { creditGrantDetails, creditGrantSourceLabel, formatCreditGrantAmount } from "../../lib/hostedCreditGrants";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { BillingPlansSection } from "./BillingPlansSection";
import { useOrganizationBillingPageModel } from "./useOrganizationBillingPageModel";

const BUY_MORE_PACK_CENTS = [5_000, 10_000, 50_000] as const;
const HOSTED_CREDIT_EXPLANATION =
  "Hosted credit pays SuperPlane-hosted machines and managed models for this organization.";
const HOSTED_CREDIT_REMAINING_CAPTION = "Remaining hosted credit";
const INVOICES_DESCRIPTION = "Recent paid invoices for this organization.";
const INVOICES_EMPTY_TITLE = "No invoices yet";
const INVOICES_EMPTY_BODY = "Paid invoices appear here after checkout.";

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
        includedRemaining={model.includedRemaining}
        purchasedRemaining={model.purchasedRemaining}
        welcomeRemaining={model.welcomeRemaining}
        currentPeriodEnd={model.currentPeriodEnd}
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
  includedRemaining,
  purchasedRemaining,
  welcomeRemaining,
  currentPeriodEnd,
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
  includedRemaining: number;
  purchasedRemaining: number;
  welcomeRemaining: number;
  currentPeriodEnd?: string;
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
      <BillingPlansSection
        canManageBilling={canManageBilling}
        creditPurchaseAllowed={creditPurchaseAllowed}
        pending={businessCheckoutPending}
        onSubscribe={onSubscribe}
      />
      <HostedCreditRemainingCard
        billingContactMessage={billingContactMessage}
        billingEnabled={billingEnabled}
        canManageBilling={canManageBilling}
        checkoutPending={checkoutPending}
        creditPurchaseAllowed={creditPurchaseAllowed}
        creditRefreshStatus={creditRefreshStatus}
        currentPeriodEnd={currentPeriodEnd}
        hasBillingCustomer={hasBillingCustomer}
        includedRemaining={includedRemaining}
        packs={packs}
        plan={plan}
        purchased={purchased}
        purchasedRemaining={purchasedRemaining}
        remaining={remaining}
        trialEndsAt={trialEndsAt}
        welcomeCreditExpiresAt={welcomeCreditExpiresAt}
        welcomeRemaining={welcomeRemaining}
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
  creditPurchaseAllowed,
  creditRefreshStatus,
  currentPeriodEnd,
  hasBillingCustomer,
  includedRemaining,
  packs,
  plan,
  purchased,
  purchasedRemaining,
  remaining,
  trialEndsAt,
  welcomeCreditExpiresAt,
  welcomeRemaining,
  onAddCredit,
}: {
  billingContactMessage?: string;
  billingEnabled: boolean;
  canManageBilling: boolean;
  checkoutPending: boolean;
  creditPurchaseAllowed: boolean;
  creditRefreshStatus: HostedCreditRefreshStatus;
  currentPeriodEnd?: string;
  hasBillingCustomer: boolean;
  includedRemaining: number;
  packs: OrganizationsHostedCreditProduct[];
  plan?: string;
  purchased: number;
  purchasedRemaining: number;
  remaining: number;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  welcomeRemaining: number;
  onAddCredit: (productId: string) => void | Promise<void>;
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
  const buckets = billingCreditBucketsView({
    welcomeRemainingCents: welcomeRemaining,
    includedRemainingCents: includedRemaining,
    purchasedRemainingCents: purchasedRemaining,
    purchasedCents: purchased,
    plan,
    trialEndsAt,
    welcomeCreditExpiresAt,
    currentPeriodEnd,
  });
  const creditRefreshMessage = hostedCreditRefreshMessage(creditRefreshStatus);
  const showBuyMore = canManageBilling && creditPurchaseAllowed;
  const remainingShares = billingCreditRemainingShares(buckets);

  return (
    <FactorySettingsCard
      title="Hosted credit"
      data-testid="billing-credit-balance"
      action={
        showBuyMore ? (
          <HostedCreditBuyMoreMenu checkoutPending={checkoutPending} packs={packs} onAddCredit={onAddCredit} />
        ) : undefined
      }
    >
      <p className="text-[12px] text-muted-foreground">{HOSTED_CREDIT_EXPLANATION}</p>
      <div className="mt-4">
        <div className="flex flex-wrap items-center gap-2">
          <p
            className="text-xl font-semibold tracking-[-0.02em] tabular-nums"
            data-testid="billing-credit-remaining-total"
          >
            {formatUsdCents(remaining)}
          </p>
          {copy.badge === "Trial" ? (
            <Badge variant="outline" className="text-muted-foreground">
              {copy.badge}
            </Badge>
          ) : null}
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground">{HOSTED_CREDIT_REMAINING_CAPTION}</p>
        <RemainingSharesBar shares={remainingShares} />
        <p className="mt-2 text-xs text-muted-foreground" data-testid="billing-credit-spend-order">
          {BILLING_SPEND_ORDER_COPY}
        </p>
      </div>
      <ul className="mt-4 divide-y divide-border">
        {buckets.map((bucket) => (
          <CreditBucketRow key={bucket.key} bucket={bucket} />
        ))}
      </ul>
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
    </FactorySettingsCard>
  );
}

function RemainingSharesBar({ shares }: { shares: BillingCreditBarShare[] }) {
  const hasRemaining = shares.some((share) => share.percent > 0);

  return (
    <div
      className="mt-3 flex h-2 overflow-hidden rounded-full bg-muted"
      data-testid="billing-credit-remaining-shares"
      role="img"
      aria-label="Remaining credit by spend order"
    >
      {hasRemaining
        ? shares
            .filter((share) => share.percent > 0)
            .map((share) => (
              <div
                key={share.key}
                className={cn("h-full", bucketAccentClassName(share.key))}
                style={{ width: `${share.percent}%` }}
              />
            ))
        : null}
    </div>
  );
}

function CreditBucketRow({ bucket }: { bucket: BillingCreditBucketView }) {
  const remainingPercent = Math.max(0, 100 - bucket.usedPercent);

  return (
    <li className="py-3 first:pt-1 last:pb-0" data-testid={`billing-credit-${bucket.key}`}>
      <div className="flex items-baseline justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn("size-1.5 shrink-0 rounded-full", bucketAccentClassName(bucket.key))} aria-hidden />
            <h3 className="text-[13px] font-medium tracking-[-0.01em]">{bucket.heading}</h3>
          </div>
          <p className="mt-0.5 pl-3.5 text-[11px] text-muted-foreground">{bucket.spendOrderLabel}</p>
        </div>
        <p
          className="shrink-0 text-[13px] font-medium tracking-[-0.01em] tabular-nums"
          data-testid={`billing-credit-${bucket.key}-remaining`}
        >
          {bucket.remainingLabel}
        </p>
      </div>
      <div
        className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-label={`${bucket.heading} remaining`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={remainingPercent}
        aria-valuetext={bucket.remainingLabel}
      >
        <div
          className={cn("h-full rounded-full", bucketAccentClassName(bucket.key))}
          style={{ width: `${remainingPercent}%` }}
        />
      </div>
      {bucket.footer ? <p className="mt-1.5 text-xs text-muted-foreground">{bucket.footer}</p> : null}
    </li>
  );
}

function bucketAccentClassName(key: BillingCreditBucketKey) {
  switch (key) {
    case "trial":
      return "bg-amber-500";
    case "included":
      return "bg-foreground";
    case "topup":
      return "bg-sky-500";
  }
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
        <Button
          type="button"
          size="sm"
          disabled={checkoutPending || !hasPurchasablePack}
          data-testid="billing-buy-more"
        >
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
        <Button
          type="button"
          size="sm"
          variant="ghost"
          disabled={portalPending}
          onClick={() => void onManageInvoices()}
        >
          {portalPending ? "Opening invoices..." : "Manage invoices"}
          <ExternalLink className="size-3.5" aria-hidden />
        </Button>
      }
    >
      <p className="text-[12px] text-muted-foreground">{INVOICES_DESCRIPTION}</p>
      {invoices.length === 0 ? <InvoiceEmptyState /> : <InvoiceList invoices={invoices} />}
    </FactorySettingsCard>
  );
}

function InvoiceEmptyState() {
  return (
    <div className="mt-4 flex flex-col items-center rounded-lg border border-dashed border-border px-6 py-8 text-center">
      <Receipt className="size-5 text-muted-foreground" aria-hidden />
      <p className="mt-2 text-sm font-medium text-foreground">{INVOICES_EMPTY_TITLE}</p>
      <p className="mt-1 text-xs text-muted-foreground">{INVOICES_EMPTY_BODY}</p>
    </div>
  );
}

function InvoiceList({ invoices }: { invoices: OrganizationsHostedCreditInvoice[] }) {
  return (
    <ul className="mt-3 divide-y divide-border">
      {invoices.map((invoice) => (
        <InvoiceRow key={invoice.id} invoice={invoice} />
      ))}
    </ul>
  );
}

function InvoiceRow({ invoice }: { invoice: OrganizationsHostedCreditInvoice }) {
  const productName = invoice.productName || "Hosted credit";
  const amount = formatUsdCents(parseWorkOrderMetric(invoice.amountCents));
  const dateLabel = formatInvoiceDate(invoice.createdAt);

  return (
    <li className="flex items-center gap-3 py-3 first:pt-1 last:pb-0">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted">
        <Receipt className="size-3.5 text-muted-foreground" aria-hidden />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium tracking-[-0.01em]">{productName}</p>
        {invoice.createdAt ? (
          <time className="mt-0.5 block text-xs text-muted-foreground" dateTime={invoice.createdAt}>
            {dateLabel}
          </time>
        ) : (
          <p className="mt-0.5 text-xs text-muted-foreground">{dateLabel}</p>
        )}
      </div>
      <p className="shrink-0 text-[13px] font-medium tracking-[-0.01em] tabular-nums">{amount}</p>
      <InvoiceStatusBadge status={invoice.status} />
    </li>
  );
}

function InvoiceStatusBadge({ status }: { status: string | undefined }) {
  const label = invoiceStatusLabel(status);
  if (status === "paid") {
    return (
      <Badge
        variant="outline"
        className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
      >
        <Check aria-hidden />
        {label}
      </Badge>
    );
  }

  return (
    <Badge variant="outline" className="text-muted-foreground">
      {label}
    </Badge>
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

function formatInvoiceDate(value: string | undefined) {
  if (!value) {
    return "—";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
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
