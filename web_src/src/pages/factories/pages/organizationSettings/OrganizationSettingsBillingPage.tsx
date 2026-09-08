import { Link, useParams } from "react-router";

import type { OrganizationsOrganizationCreditGrant } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useOrganizationCreditGrants } from "@/hooks/useOrganizationCreditGrants";
import { useOrganization } from "@/hooks/useOrganizationData";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { getApiErrorMessage } from "@/lib/errors";
import { settingsInnerMetricCardClassName } from "@/pages/organization/settings/settingsPageStyles";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import {
  creditGrantDetails,
  creditGrantSourceLabel,
  formatCreditGrantAmount,
  hostedCreditBalanceWarning,
  welcomeCreditUnusedExpiryNote,
} from "../../lib/hostedCreditGrants";
import { parseWelcomeCreditExpiresAt } from "../../lib/hostedCreditEmpty";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

export function OrganizationSettingsBillingPage() {
  const { organizationId = "", factoryKey = "" } = useParams<{ organizationId: string; factoryKey: string }>();
  const { data: organization } = useOrganization(organizationId);
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const grantsQuery = useOrganizationCreditGrants(organizationId);
  const organizationName = organization?.metadata?.name || "Organization";

  usePageTitle(["Billing", organizationName]);

  const isLoading = spend.isLoading || grantsQuery.isLoading;
  const error = spend.error ?? grantsQuery.error;
  useReportPageReady(!isLoading, { failed: Boolean(error) });

  return (
    <FactorySettingsPageFrame
      title="Billing"
      subtitle="Hosted credit pays SuperPlane-hosted models for this organization."
      actions={
        <Button type="button" disabled>
          Add hosted credit
        </Button>
      }
    >
      <BillingPageBody
        billed={parseWorkOrderMetric(spend.data?.hostedBilledCents)}
        error={error}
        factoryKey={factoryKey}
        grants={grantsQuery.data?.grants ?? []}
        isLoading={isLoading}
        organizationId={organizationId}
        purchased={parseWorkOrderMetric(spend.data?.purchasedCreditCents)}
        remaining={parseWorkOrderMetric(spend.data?.remainingCreditCents)}
        remainingCreditWarning={spend.data?.remainingCreditWarning === true}
        superplaneGrant={parseWorkOrderMetric(spend.data?.superplaneGrantCents)}
        welcomeCreditExpiresAt={spend.data?.welcomeCreditExpiresAt}
      />
    </FactorySettingsPageFrame>
  );
}

function BillingPageBody({
  billed,
  error,
  factoryKey,
  grants,
  isLoading,
  organizationId,
  purchased,
  remaining,
  remainingCreditWarning,
  superplaneGrant,
  welcomeCreditExpiresAt,
}: {
  billed: number;
  error: unknown;
  factoryKey: string;
  grants: OrganizationsOrganizationCreditGrant[];
  isLoading: boolean;
  organizationId: string;
  purchased: number;
  remaining: number;
  remainingCreditWarning: boolean;
  superplaneGrant: number;
  welcomeCreditExpiresAt?: string;
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

  const welcomeExpiresAt = parseWelcomeCreditExpiresAt(welcomeCreditExpiresAt);
  const welcomeExpired = welcomeExpiresAt != null && welcomeExpiresAt.getTime() <= Date.now();
  const warning = hostedCreditBalanceWarning(remaining, remainingCreditWarning, welcomeExpired && purchased === 0);
  const expiryNote = welcomeCreditUnusedExpiryNote({
    remainingCents: remaining,
    purchasedCents: purchased,
    welcomeCreditExpiresAt,
  });
  const spendingHref = factorySettingsSectionPath(organizationId, factoryKey, "organization", "spending");

  return (
    <>
      <FactorySettingsCard title="Hosted credit" data-testid="billing-credit-balance">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <CreditMetric label="Remaining hosted credit" value={remaining} />
          <CreditMetric label="SuperPlane grant" value={superplaneGrant} />
          <CreditMetric label="Purchased hosted credit" value={purchased} />
          <CreditMetric label="Hosted billed spend" value={billed} />
        </div>
        {warning ? <p className="mt-3 text-sm text-amber-700 dark:text-amber-400">{warning}</p> : null}
        {expiryNote ? <p className="mt-3 text-sm text-muted-foreground">{expiryNote}</p> : null}
        <p className="mt-3 text-sm text-muted-foreground">
          <Link className="underline underline-offset-2" to={spendingHref}>
            View spending
          </Link>
        </p>
      </FactorySettingsCard>

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

function CreditMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className={settingsInnerMetricCardClassName}>
      <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{label}</p>
      <p className="mt-2 text-lg font-semibold">{formatUsdCents(value)}</p>
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
