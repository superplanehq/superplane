import { ExternalLink } from "lucide-react";
import { formatUsdCents, parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import { Button } from "@/components/ui/button";
import { hostedCreditRefreshMessage, type HostedCreditRefreshStatus } from "@/lib/hostedCredit";
import { settingsInnerMetricCardClassName } from "./settingsPageStyles";

type HostedCreditProduct = {
  id?: string;
  name?: string;
  amountCents?: string | number;
};

type HostedCreditInvoice = {
  id?: string;
  createdAt?: string;
  amountCents?: string | number;
  status?: string;
  productName?: string;
};

type HostedCreditSummaryProps = {
  remainingCreditCents?: string | number;
  grantTotalCents?: string | number;
  superplaneGrantCents?: string | number;
  purchasedCreditCents?: string | number;
  hostedBilledCents?: string | number;
  remainingCreditWarning?: boolean;
  billingEnabled?: boolean;
  hasBillingCustomer?: boolean;
  canManageBilling?: boolean;
  products?: HostedCreditProduct[];
  invoices?: HostedCreditInvoice[];
  creditRefreshStatus?: HostedCreditRefreshStatus;
  checkoutPending?: boolean;
  portalPending?: boolean;
  onAddCredit?: (productId: string) => void;
  onManageInvoices?: () => void;
  cardClassName: string;
  innerCardClassName?: string;
  labelClassName: string;
  valueClassName: string;
};

export function HostedCreditSummary(props: HostedCreditSummaryProps) {
  const remaining = parseWorkOrderMetric(props.remainingCreditCents);
  const grantTotal = parseWorkOrderMetric(props.grantTotalCents);
  const superplaneGrant = parseWorkOrderMetric(props.superplaneGrantCents);
  const purchasedCredit = parseWorkOrderMetric(props.purchasedCreditCents);
  const billed = parseWorkOrderMetric(props.hostedBilledCents);
  const showGrantBreakdown = props.superplaneGrantCents != null || props.purchasedCreditCents != null;

  if (grantTotal <= 0 && superplaneGrant <= 0 && purchasedCredit <= 0 && !props.billingEnabled) {
    return null;
  }

  return (
    <HostedCreditSummaryCard
      billed={billed}
      purchasedCredit={purchasedCredit}
      remaining={remaining}
      showGrantBreakdown={showGrantBreakdown}
      superplaneGrant={superplaneGrant}
      {...props}
    />
  );
}

function HostedCreditSummaryCard({
  remaining,
  superplaneGrant,
  purchasedCredit,
  billed,
  showGrantBreakdown,
  remainingCreditWarning,
  billingEnabled = false,
  hasBillingCustomer = false,
  canManageBilling = false,
  products = [],
  invoices = [],
  creditRefreshStatus = "idle",
  checkoutPending = false,
  portalPending = false,
  onAddCredit,
  onManageInvoices,
  cardClassName,
  innerCardClassName,
  labelClassName,
  valueClassName,
}: HostedCreditSummaryProps & {
  remaining: number;
  superplaneGrant: number;
  purchasedCredit: number;
  billed: number;
  showGrantBreakdown: boolean;
}) {
  const warningMessage = hostedCreditWarning(remaining, remainingCreditWarning, billingEnabled);
  const creditRefreshMessage = hostedCreditRefreshMessage(creditRefreshStatus);
  const showBilling = showHostedBillingActions(billingEnabled, canManageBilling, products.length, hasBillingCustomer);

  return (
    <div className={cardClassName}>
      <div className={creditMetricsClassName(showGrantBreakdown, innerCardClassName)}>
        <CreditMetric
          label="Remaining hosted credit"
          labelClassName={labelClassName}
          value={remaining}
          valueClassName={valueClassName}
        />
        {showGrantBreakdown ? (
          <GrantBreakdownMetrics
            labelClassName={labelClassName}
            purchasedCredit={purchasedCredit}
            superplaneGrant={superplaneGrant}
            valueClassName={valueClassName}
          />
        ) : null}
        <CreditMetric
          label="Hosted billed spend"
          labelClassName={labelClassName}
          value={billed}
          valueClassName={valueClassName}
        />
      </div>
      {creditRefreshMessage ? (
        <p className={`mt-3 text-sm ${creditRefreshClassName(creditRefreshStatus)}`}>{creditRefreshMessage}</p>
      ) : null}
      {warningMessage ? <p className="mt-3 text-sm text-amber-700 dark:text-amber-400">{warningMessage}</p> : null}
      {showBilling ? (
        <BillingActions
          invoices={invoices}
          products={products}
          hasBillingCustomer={hasBillingCustomer}
          checkoutPending={checkoutPending}
          portalPending={portalPending}
          onAddCredit={onAddCredit}
          onManageInvoices={onManageInvoices}
        />
      ) : null}
    </div>
  );
}

function creditMetricsClassName(showGrantBreakdown: boolean, innerCardClassName?: string) {
  const grid = showGrantBreakdown ? "grid gap-3 sm:grid-cols-2 lg:grid-cols-4" : "grid gap-3 sm:grid-cols-2";
  if (!innerCardClassName) {
    return grid;
  }
  return `${grid} ${innerCardClassName}`;
}

function showHostedBillingActions(
  billingEnabled: boolean,
  canManageBilling: boolean,
  productCount: number,
  hasBillingCustomer: boolean,
) {
  if (!billingEnabled || !canManageBilling) {
    return false;
  }
  return productCount > 0 || hasBillingCustomer;
}

function GrantBreakdownMetrics({
  superplaneGrant,
  purchasedCredit,
  labelClassName,
  valueClassName,
}: {
  superplaneGrant: number;
  purchasedCredit: number;
  labelClassName: string;
  valueClassName: string;
}) {
  return (
    <>
      <CreditMetric
        label="SuperPlane grant"
        labelClassName={labelClassName}
        value={superplaneGrant}
        valueClassName={valueClassName}
      />
      <CreditMetric
        label="Purchased hosted credit"
        labelClassName={labelClassName}
        value={purchasedCredit}
        valueClassName={valueClassName}
      />
    </>
  );
}

function creditRefreshClassName(status: HostedCreditRefreshStatus) {
  if (status === "added") {
    return "text-emerald-700 dark:text-emerald-400";
  }
  return "text-gray-500 dark:text-gray-400";
}

function CreditMetric({
  label,
  value,
  labelClassName,
  valueClassName,
}: {
  label: string;
  value: number;
  labelClassName: string;
  valueClassName: string;
}) {
  return (
    <div>
      <p className={labelClassName}>{label}</p>
      <p className={valueClassName}>{formatUsdCents(value)}</p>
    </div>
  );
}

function hostedCreditWarning(
  remaining: number,
  remainingCreditWarning: boolean | undefined,
  billingEnabled: boolean,
): string | null {
  if (remaining <= 0) {
    return billingEnabled
      ? "Hosted credit is empty. Add hosted credit to start SuperPlane-hosted runs."
      : "Hosted credit is empty. SuperPlane-hosted runs cannot start until an installation admin adds credit.";
  }
  if (!remainingCreditWarning) {
    return null;
  }
  return billingEnabled
    ? "Hosted credit is low. Add hosted credit to continue SuperPlane-hosted runs."
    : "Hosted credit is low. Ask an installation admin to add credit.";
}

function BillingActions({
  products,
  invoices,
  hasBillingCustomer,
  checkoutPending,
  portalPending,
  onAddCredit,
  onManageInvoices,
}: {
  products: HostedCreditProduct[];
  invoices: HostedCreditInvoice[];
  hasBillingCustomer: boolean;
  checkoutPending: boolean;
  portalPending: boolean;
  onAddCredit?: (productId: string) => void;
  onManageInvoices?: () => void;
}) {
  const packs = products
    .slice()
    .sort((left, right) => parseWorkOrderMetric(left.amountCents) - parseWorkOrderMetric(right.amountCents));

  return (
    <div className="mt-4">
      {packs.length > 0 ? (
        <div className="grid gap-3 sm:grid-cols-3">
          {packs.map((product) => {
            const productId = product.id ?? "";
            const amount = parseWorkOrderMetric(product.amountCents);
            return (
              <div key={productId || amount} className={settingsInnerMetricCardClassName}>
                <p className="text-xs font-medium tracking-wide text-gray-500 uppercase">Hosted credit</p>
                <p className="mt-2 text-lg font-semibold">{formatUsdCents(amount)}</p>
                <Button
                  className="mt-4 w-full"
                  type="button"
                  variant="outline"
                  disabled={!productId || checkoutPending}
                  onClick={() => productId && onAddCredit?.(productId)}
                >
                  {checkoutPending ? "Opening checkout..." : "Add hosted credit"}
                </Button>
              </div>
            );
          })}
        </div>
      ) : null}
      {hasBillingCustomer ? (
        <InvoiceList invoices={invoices} portalPending={portalPending} onManageInvoices={onManageInvoices} />
      ) : packs.length > 0 ? (
        <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">Add hosted credit first to manage invoices.</p>
      ) : null}
    </div>
  );
}

function InvoiceList({
  invoices,
  portalPending,
  onManageInvoices,
}: {
  invoices: HostedCreditInvoice[];
  portalPending: boolean;
  onManageInvoices?: () => void;
}) {
  return (
    <div className="mt-4">
      <div className="flex items-center justify-between gap-3">
        <p className="workspace-section-title">Polar invoices</p>
        <Button type="button" variant="ghost" disabled={portalPending} onClick={onManageInvoices}>
          {portalPending ? "Opening invoices..." : "Manage invoices"}
          <ExternalLink className="size-3.5" aria-hidden />
        </Button>
      </div>
      {invoices.length === 0 ? (
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">No Polar invoices yet.</p>
      ) : (
        <table className="mt-2 w-full text-left text-[13px]">
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
                <td className="py-2">{formatInvoiceDate(invoice.createdAt)}</td>
                <td className="py-2">{invoice.productName || "Hosted credit"}</td>
                <td className="py-2">{formatUsdCents(parseWorkOrderMetric(invoice.amountCents))}</td>
                <td className="py-2">{invoiceStatusLabel(invoice.status)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function formatInvoiceDate(value: string | undefined) {
  if (!value) {
    return "-";
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

export type { HostedCreditProduct, HostedCreditInvoice };
