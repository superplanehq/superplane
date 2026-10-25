import { formatUsdCents } from "../../../lib/workOrderUsage";
import { factoryCardClassName } from "../../factoryPageLayoutStyles";
import type { SpendingCreditSnapshot } from "./spendingRedesignMocks";

export function SpendingKpiRow({
  metrics,
  credit,
  rangeCaption,
  billingEnabled,
}: {
  metrics: { spend: string; tokens: string; duration: string; hosted: string; byok: string };
  credit: SpendingCreditSnapshot;
  rangeCaption: string;
  billingEnabled: boolean;
}) {
  return (
    <div
      className={`grid gap-3 sm:grid-cols-2 ${billingEnabled ? "xl:grid-cols-4" : "xl:grid-cols-3"} ${factoryCardClassName} p-4`}
    >
      <SpendingKpi
        label="SuperPlane-hosted spend"
        value={metrics.hosted}
        hint={`SuperPlane-hosted models and VM time · ${rangeCaption}`}
        testId="spending-kpi-hosted"
      />
      <SpendingKpi
        label="Bring your own key spend"
        value={metrics.byok}
        hint="Estimated. SuperPlane does not bill this."
        testId="spending-kpi-byok"
      />
      <SpendingKpi label="VM time" value={metrics.duration} hint="SuperPlane runner fleets" testId="spending-kpi-vm" />
      {billingEnabled ? (
        <SpendingKpi
          label="Remaining hosted credit"
          value={formatUsdCents(credit.remainingCreditCents)}
          hint="Organization wallet"
          testId="spending-kpi-credit"
        />
      ) : null}
    </div>
  );
}

function SpendingKpi({ label, value, hint, testId }: { label: string; value: string; hint: string; testId: string }) {
  return (
    <div data-testid={testId}>
      <p className="workspace-section-label">{label}</p>
      <p className="workspace-page-title mt-1">{value}</p>
      <p className="mt-1 text-[12px] text-muted-foreground">{hint}</p>
    </div>
  );
}
