import type { OrganizationsOrganizationCreditGrant } from "@/api-client";
import { Badge } from "@/components/ui/badge";

import { creditGrantDetails, creditGrantSourceLabel, formatCreditGrantAmount } from "../../lib/hostedCreditGrants";
import { parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";

export function BillingCreditHistoryCard({ grants }: { grants: OrganizationsOrganizationCreditGrant[] }) {
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
