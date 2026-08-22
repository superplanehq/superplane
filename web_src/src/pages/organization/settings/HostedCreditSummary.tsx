import { formatUsdCents, parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";

type HostedCreditSummaryProps = {
  remainingCreditCents?: string | number;
  grantTotalCents?: string | number;
  hostedBilledCents?: string | number;
  remainingCreditWarning?: boolean;
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
  cardClassName,
  innerCardClassName,
  labelClassName,
  valueClassName,
}: HostedCreditSummaryProps) {
  const remaining = parseWorkOrderMetric(remainingCreditCents);
  const grantTotal = parseWorkOrderMetric(grantTotalCents);
  const billed = parseWorkOrderMetric(hostedBilledCents);

  if (grantTotal <= 0) {
    return null;
  }

  const warningMessage =
    remaining <= 0
      ? "Hosted credit is empty. SuperPlane-hosted runs cannot start until an installation admin adds credit."
      : remainingCreditWarning
        ? "Hosted credit is low. Ask an installation admin to add credit."
        : null;

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
      {warningMessage ? <p className="mt-3 text-sm text-amber-700 dark:text-amber-400">{warningMessage}</p> : null}
    </div>
  );
}
