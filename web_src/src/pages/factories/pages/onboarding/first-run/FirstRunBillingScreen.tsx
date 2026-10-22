import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useLocation, useSearchParams } from "react-router";
import type { OrganizationsBillingPlanOption, OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { organizationsSelectBillingPlan } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useBillingCardSetup } from "@/hooks/useBillingCardSetup";
import { syncOrganizationBilling, useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { formatUsdCents } from "../../../lib/workOrderUsage";
import { BillingCardSetup } from "../../organizationSettings/BillingCardSetup";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

function isCurrentBillingPlan(
  plan: OrganizationsBillingPlanOption | undefined,
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined,
): boolean {
  if (!plan?.key || !billing || billing.catalogKey !== plan.key) return false;
  return !plan.subscriptionRequired || (billing.planSource === "polar" && billing.creditPurchaseAllowed === true);
}

// The picker lists public plans. An active paid plan that is not listed, such
// as a private plan, still satisfies this step, so it joins the list as the
// current plan and no replacement is needed.
function currentPlanOption(
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined,
): OrganizationsBillingPlanOption | undefined {
  if (!billing?.catalogKey || billing.planCategory !== "paid" || billing.creditPurchaseAllowed !== true) {
    return undefined;
  }
  return {
    key: billing.catalogKey,
    name: billing.planName || billing.catalogKey,
    priceCents: billing.planPriceCents,
    perSeat: billing.planPerSeat,
    requirePaymentMethod: billing.requirePaymentMethod === true,
    subscriptionRequired: billing.planSource === "polar",
  };
}

function billingPlanChoices(
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined,
  currentOption: OrganizationsBillingPlanOption | undefined,
): OrganizationsBillingPlanOption[] {
  const availablePlans = billing?.availablePlans ?? [];
  if (currentOption && !availablePlans.some((option) => option.key === currentOption.key)) {
    return [currentOption, ...availablePlans];
  }
  if (billing && availablePlans.length === 0) {
    return [{ key: "none", name: "None", priceCents: "0" }];
  }
  return availablePlans;
}

function assertPlanActive(
  plan: OrganizationsBillingPlanOption,
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined,
) {
  if (!billing || billing.catalogKey !== plan.key)
    throw new Error("The plan is not active yet. Try again in a moment.");
  if (plan.requirePaymentMethod && !billing.hasPaymentMethod) throw new Error("Add a payment card before continuing.");
  if (plan.subscriptionRequired && !(billing.planSource === "polar" && billing.creditPurchaseAllowed)) {
    throw new Error("Complete subscription checkout before continuing.");
  }
}

function useOnboardingBilling(organizationId: string, onContinue: () => Promise<void>) {
  const billing = useOrganizationBilling(organizationId);
  const cardSetup = useBillingCardSetup(organizationId);
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const [requestedPlanKey, setRequestedPlanKey] = useState(searchParams.get("plan") ?? "");
  const [error, setError] = useState("");
  const [checkoutReturned, setCheckoutReturned] = useState(searchParams.get("subscribed") === "1");
  const currentOption = currentPlanOption(billing.data);
  const plans = billingPlanChoices(billing.data, currentOption);
  const noPlans = billing.isSuccess && plans[0]?.key === "none";
  const selectedKey = noPlans ? "none" : requestedPlanKey;
  const plan = plans.find((option) => option.key === selectedKey);
  const currentPlan = isCurrentBillingPlan(plan, billing.data);
  const needsCard =
    plan?.requirePaymentMethod && (!plan.subscriptionRequired || currentPlan) && !billing.data?.hasPaymentMethod;
  const subscriptionActive = billing.data?.planSource === "polar" && billing.data.creditPurchaseAllowed;
  const refresh = useMutation({
    mutationFn: async () => {
      await syncOrganizationBilling(organizationId);
      const result = await billing.refetch();
      if (result.error) throw result.error;
    },
    onError: (cause) => setError(getApiErrorMessage(cause, "Could not verify billing. Try again.")),
  });
  useEffect(() => {
    if (!checkoutReturned) return;
    setCheckoutReturned(false);
    refresh.mutate();
  }, [checkoutReturned, refresh]);
  useEffect(() => {
    if (selectedKey || !currentOption?.key) return;
    setRequestedPlanKey(currentOption.key);
  }, [currentOption?.key, selectedKey]);

  const activate = useMutation({
    mutationFn: async (plan: OrganizationsBillingPlanOption) => {
      setError("");
      if (!plan?.key) return;
      if (searchParams.has("subscribed")) await syncOrganizationBilling(organizationId);
      const verified = await billing.refetch();
      if (verified.error) throw verified.error;
      if (!verified.data) throw new Error("Could not verify billing. Try again.");
      if (plan.requirePaymentMethod && !plan.subscriptionRequired && !verified.data.hasPaymentMethod) {
        throw new Error("Add a payment card before continuing.");
      }
      if (searchParams.has("subscribed")) assertPlanActive(plan, verified.data);
      if (!isCurrentBillingPlan(plan, verified.data)) {
        const response = await organizationsSelectBillingPlan(
          withOrganizationHeader({
            organizationId,
            path: { id: organizationId },
            body: {
              planKey: plan.key,
              returnPath: `${location.pathname}?step=billing&plan=${encodeURIComponent(plan.key)}`,
            },
          }),
        );
        if (response.data?.checkoutUrl) {
          window.location.assign(response.data.checkoutUrl);
          return;
        }
        if (plan.subscriptionRequired) throw new Error("Could not start subscription checkout. Try again.");
      }
      const updated = await billing.refetch();
      if (updated.error) throw updated.error;
      assertPlanActive(plan, updated.data);
      await onContinue();
    },
    onError: (cause) => setError(getApiErrorMessage(cause, "Could not select the plan. Try again.")),
  });
  return {
    billing,
    plans,
    noPlans,
    plan,
    currentOption,
    currentPlan,
    selectedKey,
    selectPlan: (option: OrganizationsBillingPlanOption) => {
      setRequestedPlanKey(option.key ?? "");
      setError("");
      if (option.subscriptionRequired && !isCurrentBillingPlan(option, billing.data)) {
        activate.mutate(option);
        return;
      }
      if (option.requirePaymentMethod && !billing.data?.hasPaymentMethod) cardSetup.open();
    },
    cardSetup,
    needsCard,
    subscriptionActive,
    activate,
    refresh,
    error,
  };
}

function BillingPlanChoice({
  plan,
  selected,
  current,
  disabled,
  onSelect,
}: {
  plan: OrganizationsBillingPlanOption;
  selected: boolean;
  current: boolean;
  disabled: boolean;
  onSelect: () => void;
}) {
  return (
    <FirstRunPanel>
      <Button
        type="button"
        variant="ghost"
        className="h-auto w-full flex-col items-start gap-1 rounded-md whitespace-normal text-left aria-pressed:bg-accent aria-pressed:ring-1 aria-pressed:ring-ring"
        aria-pressed={selected}
        disabled={disabled}
        onClick={onSelect}
      >
        <span className="font-medium">{plan.name}</span>
        <span>
          {formatUsdCents(Number(plan.priceCents ?? 0))}
          {plan.perSeat && " per seat"}
        </span>
        <span className="text-sm text-muted-foreground">
          {current
            ? "Current plan"
            : plan.subscriptionRequired
              ? "Subscription checkout required"
              : plan.requirePaymentMethod
                ? "Card required. No subscription."
                : "No card or subscription required."}
        </span>
      </Button>
    </FirstRunPanel>
  );
}

export function FirstRunBillingScreen({
  organizationId,
  chrome,
  sphere,
  saving,
  onContinue,
}: {
  organizationId: string;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  saving: boolean;
  onContinue: () => Promise<void>;
}) {
  const {
    billing,
    plans,
    noPlans,
    plan,
    currentOption,
    currentPlan,
    selectedKey,
    selectPlan,
    cardSetup,
    needsCard,
    subscriptionActive,
    activate,
    refresh,
    error,
  } = useOnboardingBilling(organizationId, onContinue);
  const busy = saving || activate.isPending || refresh.isPending;
  return (
    <FirstRunShell testId="first-run-billing" chrome={chrome} busy={busy} sphere={sphere} visual="preview" width="wide">
      <FirstRunHeading headline="Choose a plan">
        <p>
          {noPlans
            ? "No plans are available. Finish setup with no plan."
            : "Select a plan for your organization to finish setup."}
        </p>
      </FirstRunHeading>
      <div className="mt-8 space-y-4">
        {billing.isPending && <p role="status">Loading plans…</p>}
        {billing.isError && <p role="alert">Could not load plans. Try again.</p>}
        {plans.map((option) => (
          <BillingPlanChoice
            key={option.key}
            plan={option}
            selected={selectedKey === option.key}
            current={currentOption?.key !== undefined && option.key === currentOption.key}
            disabled={
              busy || cardSetup.isPending || Boolean(subscriptionActive && option.key !== billing.data?.catalogKey)
            }
            onSelect={() => selectPlan(option)}
          />
        ))}
        {needsCard && <BillingCardSetup setup={cardSetup} disabled={busy} />}
        <LoadingButton
          type="button"
          className="w-full"
          loading={busy}
          loadingText="Saving plan…"
          disabled={!plan || Boolean(needsCard) || billing.isPending || billing.isError}
          onClick={() => {
            if (plan) activate.mutate(plan);
          }}
        >
          {plan?.subscriptionRequired && !currentPlan ? `Subscribe to ${plan.name}` : "Finish setup"}
        </LoadingButton>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </div>
    </FirstRunShell>
  );
}
