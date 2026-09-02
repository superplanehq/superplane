import { usePageTitle } from "@/hooks/usePageTitle";
import { useOrganization, useOrganizationUsers } from "@/hooks/useOrganizationData";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useHostedCreditReturnRefresh } from "@/hooks/useHostedCreditReturnRefresh";
import {
  useCreateBillingPortalSession,
  useCreateHostedCreditCheckout,
  useHostedCreditProducts,
} from "@/hooks/useLLMModelAllowlists";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { clearHostedCreditGrantSnapshot, rememberHostedCreditGrantSnapshot } from "@/lib/hostedCredit";
import { hostedCreditOwnerContactCopy } from "@/lib/hostedCreditOwnerContact";
import { showErrorToast } from "@/lib/toast";
import { parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { useParams, useSearchParams } from "react-router";
import { BYOKModelsCard } from "@/pages/organization/settings/BYOKModelsCard";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";
import { factoryCardClassName } from "../factoryPageLayoutStyles";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import {
  WorkspaceUsageByMachineTypeTable,
  WorkspaceUsageByModelTable,
  WorkspaceUsageTotals,
} from "../settings/WorkspaceUsageBreakdown";

export function OrganizationSettingsWorkspaceUsagePage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  usePageTitle(["Spending"]);
  const { data, isLoading, error, refetch } = useOrganizationWorkspaceUsage(organizationId || "");
  const [searchParams] = useSearchParams();
  const creditAdded = searchParams.get("credit") === "added";

  return (
    <FactorySettingsPageFrame
      title="Spending"
      subtitle="Review factory token usage, VM time, and estimated spend for this organization."
    >
      <WorkspaceUsageBody
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

function WorkspaceUsageBody({
  organizationId,
  data,
  isLoading,
  error,
  creditAdded,
  refetch,
}: {
  organizationId: string;
  data: ReturnType<typeof useOrganizationWorkspaceUsage>["data"];
  isLoading: boolean;
  error: unknown;
  creditAdded: boolean;
  refetch: ReturnType<typeof useOrganizationWorkspaceUsage>["refetch"];
}) {
  if (isLoading) {
    return (
      <FactorySettingsCard>
        <p className="text-[13px] text-muted-foreground">Loading workspace usage...</p>
      </FactorySettingsCard>
    );
  }
  if (error || !data) {
    return (
      <FactorySettingsCard>
        <p className="text-[13px] text-destructive">Unable to load workspace usage.</p>
      </FactorySettingsCard>
    );
  }

  return (
    <WorkspaceUsageLoaded creditAdded={creditAdded} data={data} organizationId={organizationId} refetch={refetch} />
  );
}

function WorkspaceUsageLoaded({
  organizationId,
  data,
  creditAdded,
  refetch,
}: {
  organizationId: string;
  data: NonNullable<ReturnType<typeof useOrganizationWorkspaceUsage>["data"]>;
  creditAdded: boolean;
  refetch: ReturnType<typeof useOrganizationWorkspaceUsage>["refetch"];
}) {
  const { canAct } = usePermissions();
  const canManageBilling = canAct("org", "update");
  const grantTotalCents = parseWorkOrderMetric(data.grantTotalCents);
  const billing = useHostedCreditActions(organizationId, data.billingEnabled === true, grantTotalCents);
  const creditRefreshStatus = useHostedCreditReturnRefresh({
    organizationId,
    creditAdded,
    grantTotalCents,
    refetch,
  });
  const billingContactMessage = useHostedCreditOwnerContactMessage(
    organizationId,
    data.billingEnabled === true && !canManageBilling,
  );

  return (
    <div className="flex flex-col gap-4">
      <WorkspaceUsageTotals
        periodDays={data.periodDays ?? 30}
        totalTokens={parseWorkOrderMetric(data.totalTokens)}
        totalCostCents={parseWorkOrderMetric(data.totalCostCents)}
        totalDurationSeconds={parseWorkOrderMetric(data.totalDurationSeconds)}
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
        canManageBilling={canManageBilling}
        billingContactMessage={billingContactMessage}
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
      <WorkspaceUsageByModelTable byModel={data.byModel ?? []} />
      <WorkspaceUsageByMachineTypeTable byMachineType={data.byMachineType ?? []} />
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

/**
 * Builds the "contact an owner" sentence shown to admins and viewers in
 * place of the hosted credit checkout packs. Only fetched when the caller
 * actually needs it (Polar is on and the signed-in user cannot manage
 * billing); owners never see the extra requests.
 */
function useHostedCreditOwnerContactMessage(organizationId: string, needed: boolean): string | undefined {
  const { data: organization } = useOrganization(organizationId, needed);
  const { data: users = [] } = useOrganizationUsers(organizationId, true, needed);

  if (!needed) {
    return undefined;
  }

  const owners = users
    .filter((user) => user.status?.roles?.some((role) => role.roleName === "org_owner"))
    .map((user) => ({ name: user.spec?.displayName, email: user.metadata?.email }));

  return hostedCreditOwnerContactCopy({
    organizationName: organization?.metadata?.name,
    owners,
  });
}
