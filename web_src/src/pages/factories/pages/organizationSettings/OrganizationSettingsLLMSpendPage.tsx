import { usePageTitle } from "@/hooks/usePageTitle";
import { useOrganizationLLMSpend } from "@/hooks/useOrganizationLLMSpend";
import { useHostedCreditReturnRefresh } from "@/hooks/useHostedCreditReturnRefresh";
import {
  useCreateBillingPortalSession,
  useCreateHostedCreditCheckout,
  useHostedCreditProducts,
} from "@/hooks/useLLMModelAllowlists";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { clearHostedCreditGrantSnapshot, rememberHostedCreditGrantSnapshot } from "@/lib/hostedCredit";
import { showErrorToast } from "@/lib/toast";
import { parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { useParams, useSearchParams } from "react-router";
import { BYOKModelsCard } from "@/pages/organization/settings/BYOKModelsCard";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";
import { factoryCardClassName } from "../factoryPageLayoutStyles";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { LLMUsageByModelTable, LLMUsageTotals } from "../settings/LLMUsageBreakdown";

export function OrganizationSettingsLLMSpendPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  usePageTitle(["LLM spend"]);
  const { data, isLoading, error, refetch } = useOrganizationLLMSpend(organizationId || "");
  const [searchParams] = useSearchParams();
  const creditAdded = searchParams.get("credit") === "added";

  return (
    <FactorySettingsPageFrame
      title="LLM spend"
      subtitle="Review factory token usage and estimated model cost for this organization."
    >
      <LLMSpendBody
        creditAdded={creditAdded}
        data={data}
        error={error}
        isLoading={isLoading}
        organizationId={organizationId || ""}
        refetch={refetch}
      />
    </FactorySettingsPageFrame>
  );
}

function LLMSpendBody({
  organizationId,
  data,
  isLoading,
  error,
  creditAdded,
  refetch,
}: {
  organizationId: string;
  data: ReturnType<typeof useOrganizationLLMSpend>["data"];
  isLoading: boolean;
  error: unknown;
  creditAdded: boolean;
  refetch: ReturnType<typeof useOrganizationLLMSpend>["refetch"];
}) {
  if (isLoading) {
    return (
      <FactorySettingsCard>
        <p className="text-[13px] text-muted-foreground">Loading LLM spend...</p>
      </FactorySettingsCard>
    );
  }
  if (error || !data) {
    return (
      <FactorySettingsCard>
        <p className="text-[13px] text-destructive">Unable to load LLM spend.</p>
      </FactorySettingsCard>
    );
  }

  return <LLMSpendLoaded creditAdded={creditAdded} data={data} organizationId={organizationId} refetch={refetch} />;
}

function LLMSpendLoaded({
  organizationId,
  data,
  creditAdded,
  refetch,
}: {
  organizationId: string;
  data: NonNullable<ReturnType<typeof useOrganizationLLMSpend>["data"]>;
  creditAdded: boolean;
  refetch: ReturnType<typeof useOrganizationLLMSpend>["refetch"];
}) {
  const { canAct } = usePermissions();
  const grantTotalCents = parseWorkOrderMetric(data.grantTotalCents);
  const billing = useHostedCreditActions(organizationId, data.billingEnabled === true, grantTotalCents);
  const creditRefreshStatus = useHostedCreditReturnRefresh({
    organizationId,
    creditAdded,
    grantTotalCents,
    refetch,
  });

  return (
    <div className="flex flex-col gap-4">
      <LLMUsageTotals
        periodDays={data.periodDays ?? 30}
        totalTokens={parseWorkOrderMetric(data.totalTokens)}
        totalCostCents={parseWorkOrderMetric(data.totalCostCents)}
      />
      <HostedCreditSummary
        remainingCreditCents={data.remainingCreditCents}
        grantTotalCents={data.grantTotalCents}
        superplaneGrantCents={data.superplaneGrantCents}
        purchasedCreditCents={data.purchasedCreditCents}
        hostedBilledCents={data.hostedBilledCents}
        remainingCreditWarning={data.remainingCreditWarning}
        billingEnabled={data.billingEnabled}
        hasBillingCustomer={data.hasBillingCustomer}
        canManageBilling={canAct("org", "update")}
        products={billing.products}
        invoices={data.invoices}
        creditRefreshStatus={creditRefreshStatus}
        checkoutPending={billing.checkoutPending}
        portalPending={billing.portalPending}
        onAddCredit={billing.startCheckout}
        onManageInvoices={billing.openInvoices}
        cardClassName={`${factoryCardClassName} p-4`}
        labelClassName="workspace-section-label"
        valueClassName="workspace-page-title mt-1"
      />
      <BYOKModelsCard organizationId={organizationId} />
      <LLMUsageByModelTable byModel={data.byModel ?? []} />
    </div>
  );
}

function useHostedCreditActions(organizationId: string, billingEnabled: boolean, grantTotalCents: number) {
  const productsQuery = useHostedCreditProducts(organizationId, billingEnabled);
  const checkout = useCreateHostedCreditCheckout(organizationId);
  const portal = useCreateBillingPortalSession(organizationId);

  return {
    products: productsQuery.data?.products ?? [],
    checkoutPending: checkout.isPending,
    portalPending: portal.isPending,
    startCheckout: async (productId: string) => {
      rememberHostedCreditGrantSnapshot(organizationId, grantTotalCents);
      try {
        const url = await checkout.mutateAsync(productId);
        window.location.assign(url);
      } catch (checkoutError) {
        clearHostedCreditGrantSnapshot(organizationId);
        showErrorToast(getApiErrorMessage(checkoutError, "Unable to start checkout."));
      }
    },
    openInvoices: async () => {
      try {
        const url = await portal.mutateAsync();
        window.location.assign(url);
      } catch (portalError) {
        showErrorToast(getApiErrorMessage(portalError, "Add hosted credit first."));
      }
    },
  };
}
