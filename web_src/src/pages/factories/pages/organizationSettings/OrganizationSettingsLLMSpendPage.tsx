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

  useEffect(() => {
    if (creditAdded) {
      void refetch();
    }
  }, [creditAdded, refetch]);

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
}: {
  organizationId: string;
  data: ReturnType<typeof useOrganizationLLMSpend>["data"];
  isLoading: boolean;
  error: unknown;
  creditAdded: boolean;
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

  return <LLMSpendLoaded creditAdded={creditAdded} data={data} organizationId={organizationId} />;
}

function LLMSpendLoaded({
  organizationId,
  data,
  creditAdded,
}: {
  organizationId: string;
  data: NonNullable<ReturnType<typeof useOrganizationLLMSpend>["data"]>;
  creditAdded: boolean;
}) {
  const { canAct } = usePermissions();
  const billing = useHostedCreditActions(organizationId, data.billingEnabled === true);

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
        hostedBilledCents={data.hostedBilledCents}
        remainingCreditWarning={data.remainingCreditWarning}
        billingEnabled={data.billingEnabled}
        hasBillingCustomer={data.hasBillingCustomer}
        canManageBilling={canAct("org", "update")}
        products={billing.products}
        creditAdded={creditAdded}
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

function useHostedCreditActions(organizationId: string, billingEnabled: boolean) {
  const productsQuery = useHostedCreditProducts(organizationId, billingEnabled);
  const checkout = useCreateHostedCreditCheckout(organizationId);
  const portal = useCreateBillingPortalSession(organizationId);

  return {
    products: productsQuery.data?.products ?? [],
    checkoutPending: checkout.isPending,
    portalPending: portal.isPending,
    startCheckout: async (productId: string) => {
      try {
        const url = await checkout.mutateAsync(productId);
        window.location.assign(url);
      } catch (checkoutError) {
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
