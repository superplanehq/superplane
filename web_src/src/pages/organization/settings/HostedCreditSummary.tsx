import { formatUsdCents, parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import { Button } from "@/components/ui/button";
import { settingsInnerMetricCardClassName } from "./settingsPageStyles";

type HostedCreditProduct = {
  id?: string;
  name?: string;
  amountCents?: string | number;
};

type HostedCreditSummaryProps = {
  remainingCreditCents?: string | number;
  grantTotalCents?: string | number;
  hostedBilledCents?: string | number;
  remainingCreditWarning?: boolean;
  billingEnabled?: boolean;
  hasBillingCustomer?: boolean;
  canManageBilling?: boolean;
  products?: HostedCreditProduct[];
  creditAdded?: boolean;
  checkoutPending?: boolean;
  portalPending?: boolean;
  onAddCredit?: (productId: string) => void;
  onManageInvoices?: () => void;
  cardClassName: string;
  innerCardClassName?: string;
  labelClassName: string;
  valueClassName: string;
};

export function HostedCreditSummary({
  remainingCreditCents,
  grantTotalCents,
  hostedBilledCents,
  remainingCreditWarning,
  billingEnabled = false,
  hasBillingCustomer = false,
  canManageBilling = false,
  products = [],
  creditAdded = false,
  checkoutPending = false,
  portalPending = false,
  onAddCredit,
  onManageInvoices,
  cardClassName,
  innerCardClassName,
  labelClassName,
  valueClassName,
}: HostedCreditSummaryProps) {
  const remaining = parseWorkOrderMetric(remainingCreditCents);
  const grantTotal = parseWorkOrderMetric(grantTotalCents);
  const billed = parseWorkOrderMetric(hostedBilledCents);

  if (grantTotal <= 0 && !billingEnabled) {
    return null;
  }

  const warningMessage = hostedCreditWarning(remaining, remainingCreditWarning, billingEnabled);

  return (
    <div className={cardClassName}>
      <div
        className={innerCardClassName ? `grid gap-3 sm:grid-cols-3 ${innerCardClassName}` : "grid gap-3 sm:grid-cols-3"}
      >
        <div>
          <p className={labelClassName}>Remaining hosted credit</p>
          <p className={valueClassName}>{formatUsdCents(remaining)}</p>
        </div>
        <div>
          <p className={labelClassName}>Grant total</p>
          <p className={valueClassName}>{formatUsdCents(grantTotal)}</p>
        </div>
        <div>
          <p className={labelClassName}>Hosted billed spend</p>
          <p className={valueClassName}>{formatUsdCents(billed)}</p>
        </div>
      </div>
      {creditAdded ? (
        <p className="mt-3 text-sm text-emerald-700 dark:text-emerald-400">
          Hosted credit was added. Refreshing totals.
        </p>
      ) : null}
      {warningMessage ? <p className="mt-3 text-sm text-amber-700 dark:text-amber-400">{warningMessage}</p> : null}
      {billingEnabled && canManageBilling && (products.length > 0 || hasBillingCustomer) ? (
        <BillingActions
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
  hasBillingCustomer,
  checkoutPending,
  portalPending,
  onAddCredit,
  onManageInvoices,
}: {
  products: HostedCreditProduct[];
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
        <Button className="mt-3" type="button" variant="ghost" disabled={portalPending} onClick={onManageInvoices}>
          {portalPending ? "Opening invoices..." : "Manage invoices"}
        </Button>
      ) : packs.length > 0 ? (
        <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">Add hosted credit first to manage invoices.</p>
      ) : null}
    </div>
  );
}

export type { HostedCreditProduct };
