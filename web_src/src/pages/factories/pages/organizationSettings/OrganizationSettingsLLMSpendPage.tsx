import { useEffect } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useOrganizationLLMSpend } from "@/hooks/useOrganizationLLMSpend";
import {
  useCreateBillingPortalSession,
  useCreateHostedCreditCheckout,
  useHostedCreditProducts,
} from "@/hooks/useLLMModelAllowlists";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { useParams, useSearchParams } from "react-router";
import { BYOKModelsCard } from "@/pages/organization/settings/BYOKModelsCard";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";
import { factoryCardClassName } from "../factoryPageLayoutStyles";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

export function OrganizationSettingsLLMSpendPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  usePageTitle(["LLM spend"]);
  const { data, isLoading, error, refetch } = useOrganizationLLMSpend(organizationId || "");
  const { canAct } = usePermissions();
  const canManageBilling = canAct("org", "update");
  const [searchParams] = useSearchParams();
  const creditAdded = searchParams.get("credit") === "added";
  const productsQuery = useHostedCreditProducts(organizationId || "", data?.billingEnabled === true);
  const checkout = useCreateHostedCreditCheckout(organizationId || "");
  const portal = useCreateBillingPortalSession(organizationId || "");

  useEffect(() => {
    if (creditAdded) {
      void refetch();
    }
  }, [creditAdded, refetch]);

  const totalTokens = parseWorkOrderMetric(data?.totalTokens);
  const totalCostCents = parseWorkOrderMetric(data?.totalCostCents);
  const periodDays = data?.periodDays ?? 30;
  const byModel = data?.byModel ?? [];

  return (
    <FactorySettingsPageFrame
      title="LLM spend"
      subtitle="Review factory token usage and estimated model cost for this organization."
    >
      {isLoading ? (
        <FactorySettingsCard>
          <p className="text-[13px] text-muted-foreground">Loading LLM spend...</p>
        </FactorySettingsCard>
      ) : error ? (
        <FactorySettingsCard>
          <p className="text-[13px] text-destructive">Unable to load LLM spend.</p>
        </FactorySettingsCard>
      ) : (
        <div className="flex flex-col gap-4">
          <div className={`grid gap-3 sm:grid-cols-2 ${factoryCardClassName} p-4`}>
            <div>
              <p className="workspace-section-label">Tokens · last {periodDays} days</p>
              <p className="workspace-page-title mt-1">{formatCompactTokens(totalTokens)}</p>
            </div>
            <div>
              <p className="workspace-section-label">Estimated spend</p>
              <p className="workspace-page-title mt-1">{formatUsdCents(totalCostCents)}</p>
            </div>
          </div>
          <HostedCreditSummary
            remainingCreditCents={data?.remainingCreditCents}
            grantTotalCents={data?.grantTotalCents}
            hostedBilledCents={data?.hostedBilledCents}
            remainingCreditWarning={data?.remainingCreditWarning}
            billingEnabled={data?.billingEnabled}
            hasBillingCustomer={data?.hasBillingCustomer}
            canManageBilling={canManageBilling}
            products={productsQuery.data?.products ?? []}
            creditAdded={creditAdded}
            checkoutPending={checkout.isPending}
            portalPending={portal.isPending}
            onAddCredit={async (productId) => {
              try {
                const url = await checkout.mutateAsync(productId);
                window.location.assign(url);
              } catch (checkoutError) {
                showErrorToast(getApiErrorMessage(checkoutError, "Unable to start checkout."));
              }
            }}
            onManageInvoices={async () => {
              try {
                const url = await portal.mutateAsync();
                window.location.assign(url);
              } catch (portalError) {
                showErrorToast(getApiErrorMessage(portalError, "Add hosted credit first."));
              }
            }}
            cardClassName={`${factoryCardClassName} p-4`}
            labelClassName="workspace-section-label"
            valueClassName="workspace-page-title mt-1"
          />
          <BYOKModelsCard organizationId={organizationId || ""} />
          <div className={factoryCardClassName}>
            <p className="workspace-section-title px-4 pt-4">By model</p>
            {byModel.length === 0 ? (
              <p className="px-4 pb-4 pt-2 text-[13px] text-muted-foreground">
                No factory LLM usage is recorded for this period.
              </p>
            ) : (
              <table className="mt-2 w-full text-left text-[13px]">
                <thead>
                  <tr className="border-b border-border text-muted-foreground">
                    <th className="px-4 py-2 font-medium">Provider</th>
                    <th className="px-4 py-2 font-medium">Model</th>
                    <th className="px-4 py-2 font-medium">Tokens</th>
                    <th className="px-4 py-2 font-medium">Spend</th>
                  </tr>
                </thead>
                <tbody>
                  {byModel.map((row) => (
                    <tr key={`${row.provider}-${row.model}`} className="border-b border-border last:border-0">
                      <td className="px-4 py-2">{row.provider}</td>
                      <td className="px-4 py-2">{row.model}</td>
                      <td className="px-4 py-2">{formatCompactTokens(parseWorkOrderMetric(row.totalTokens))}</td>
                      <td className="px-4 py-2">{formatUsdCents(parseWorkOrderMetric(row.costCents))}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </FactorySettingsPageFrame>
  );
}
