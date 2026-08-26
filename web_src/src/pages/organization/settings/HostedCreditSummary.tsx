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

export function HostedCreditSummary(props: HostedCreditSummaryProps) {
  const remaining = parseWorkOrderMetric(props.remainingCreditCents);
  const grantTotal = parseWorkOrderMetric(props.grantTotalCents);
  const billed = parseWorkOrderMetric(props.hostedBilledCents);

  if (grantTotal <= 0 && !props.billingEnabled) {
    return null;
  }

  return <HostedCreditSummaryCard billed={billed} grantTotal={grantTotal} remaining={remaining} {...props} />;
}

function HostedCreditSummaryCard({
  remaining,
  grantTotal,
  billed,
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
}: HostedCreditSummaryProps & { remaining: number; grantTotal: number; billed: number }) {
  const warningMessage = hostedCreditWarning(remaining, remainingCreditWarning, billingEnabled);
  const showBillingActions = Boolean(billingEnabled && canManageBilling && (products.length > 0 || hasBillingCustomer));

  return (
    <div className={cardClassName}>
      <div
        className={innerCardClassName ? `grid gap-3 sm:grid-cols-3 ${innerCardClassName}` : "grid gap-3 sm:grid-cols-3"}
      >
        <CreditMetric
          label="Remaining hosted credit"
          labelClassName={labelClassName}
          value={remaining}
          valueClassName={valueClassName}
        />
        <CreditMetric
          label="Grant total"
          labelClassName={labelClassName}
          value={grantTotal}
          valueClassName={valueClassName}
        />
        <CreditMetric
          label="Hosted billed spend"
          labelClassName={labelClassName}
          value={billed}
          valueClassName={valueClassName}
        />
      </div>
      {creditAdded ? (
        <p className="mt-3 text-sm text-emerald-700 dark:text-emerald-400">
          Hosted credit was added. Refreshing totals.
        </p>
      ) : null}
      {warningMessage ? <p className="mt-3 text-sm text-amber-700 dark:text-amber-400">{warningMessage}</p> : null}
      {showBillingActions ? (
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
