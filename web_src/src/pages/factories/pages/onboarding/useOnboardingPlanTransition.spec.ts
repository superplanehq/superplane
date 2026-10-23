import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { useOnboardingPlanTransition } from "./useOnboardingPlanTransition";

let newBilling = true;
let billing: OrganizationsDescribeOrganizationBillingResponse;
let lookupFailed = false;
const verify = vi.fn(async () => true);
vi.mock("@/hooks/useBYOKAccess", () => ({ useBYOKAccess: () => ({ allowed: true, ready: true, verify }) }));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ organizationReady: true, isLoading: false, has: () => newBilling }),
}));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  useOrganizationBilling: () => ({
    data: billing,
    refetch: async () => (lookupFailed ? { error: new Error("unavailable") } : { data: billing }),
  }),
}));
const toast = vi.fn();
vi.mock("@/lib/toast", () => ({ showErrorToast: toast }));
beforeEach(() => {
  newBilling = true;
  lookupFailed = false;
  billing = { billingEnabled: true, availablePlans: [{ key: "team", name: "Team" }] };
  verify.mockReset();
  verify.mockResolvedValue(true);
  toast.mockClear();
});
function mount() {
  const finish = vi.fn(async () => {});
  const goToScreen = vi.fn();
  const hook = renderHook(() =>
    useOnboardingPlanTransition({ saving: false, finish }, { screen: "tickets", goToScreen }, "org"),
  );
  return { ...hook, finish, goToScreen };
}

it("opens the plan step before model access is resolved", async () => {
  const { result, finish, goToScreen } = mount();
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("billing"));
  expect(verify).not.toHaveBeenCalled();
  expect(finish).not.toHaveBeenCalled();
});
it.each([true, false])("uses freshly verified BYOK access after plan activation: %s", async (allowed) => {
  const { result, finish, goToScreen } = mount();
  verify.mockResolvedValue(allowed);
  await act(async () => result.current.continueAfterPlan());
  if (allowed) {
    expect(goToScreen).toHaveBeenCalledWith("agent");
    expect(finish).not.toHaveBeenCalled();
    return;
  }
  expect(finish).toHaveBeenCalledTimes(1);
  expect(goToScreen).not.toHaveBeenCalled();
});
it("keeps setup open when product permission cannot be verified", async () => {
  const { result, finish, goToScreen } = mount();
  verify.mockRejectedValue(new Error("Could not verify permissions"));
  await act(async () => result.current.continueAfterPlan());
  expect(finish).not.toHaveBeenCalled();
  expect(goToScreen).not.toHaveBeenCalled();
  expect(toast).toHaveBeenCalled();
});
it("keeps ticket setup open when billing lookup fails", async () => {
  lookupFailed = true;
  const { result, finish, goToScreen } = mount();
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(toast).toHaveBeenCalled());
  expect(finish).not.toHaveBeenCalled();
  expect(goToScreen).not.toHaveBeenCalled();
  expect(verify).not.toHaveBeenCalled();
});
it.each(["rollout-off", "billing-off", "no-plans"])("omits the plan step for %s", async (condition) => {
  if (condition === "rollout-off") newBilling = false;
  if (condition === "billing-off") billing.billingEnabled = false;
  if (condition === "no-plans") billing.availablePlans = [];
  const { result, goToScreen } = mount();
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("agent"));
  expect(goToScreen).not.toHaveBeenCalledWith("billing");
});

const activePlans: Record<string, Partial<OrganizationsDescribeOrganizationBillingResponse>> = {
  "system $0 plan": { planCategory: "paid", creditPurchaseAllowed: true, planSource: "system" },
  "admin plan": { planCategory: "paid", creditPurchaseAllowed: true, planSource: "admin" },
  subscription: { planCategory: "paid", creditPurchaseAllowed: true, planSource: "polar" },
  "plan with card on file": {
    planCategory: "paid",
    creditPurchaseAllowed: true,
    requirePaymentMethod: true,
    hasPaymentMethod: true,
  },
};
const inactivePlans: Record<string, Partial<OrganizationsDescribeOrganizationBillingResponse>> = {
  "no plan": { planCategory: "none", creditPurchaseAllowed: false },
  "lapsed trial or ended subscription": { planCategory: "paid", creditPurchaseAllowed: false },
  "card required without card": {
    planCategory: "paid",
    creditPurchaseAllowed: true,
    requirePaymentMethod: true,
    hasPaymentMethod: false,
  },
};

it.each(Object.keys(activePlans))("skips the plan step for an active plan: %s", async (name) => {
  Object.assign(billing, activePlans[name]);
  const { result, goToScreen } = mount();
  expect(result.current.hasBillingStep).toBe(false);
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("agent"));
  expect(goToScreen).not.toHaveBeenCalledWith("billing");
});
it("finishes setup for an active plan when own-key access is not verified", async () => {
  Object.assign(billing, activePlans["admin plan"]);
  verify.mockResolvedValue(false);
  const { result, finish, goToScreen } = mount();
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
  expect(goToScreen).not.toHaveBeenCalled();
});
it.each(Object.keys(inactivePlans))("keeps the plan step for an inactive plan: %s", async (name) => {
  Object.assign(billing, inactivePlans[name]);
  const { result, goToScreen } = mount();
  expect(result.current.hasBillingStep).toBe(true);
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("billing"));
  expect(goToScreen).not.toHaveBeenCalledWith("agent");
});
it("keeps the plan step after the user activates a plan on it", async () => {
  const { result, rerender, goToScreen } = mount();
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("billing"));
  Object.assign(billing, activePlans["subscription"]);
  rerender();
  expect(result.current.hasBillingStep).toBe(true);
});
it("keeps the plan step after a card save on a restored visit", () => {
  Object.assign(billing, inactivePlans["card required without card"]);
  const finish = vi.fn(async () => {});
  const goToScreen = vi.fn();
  const { result, rerender } = renderHook(() =>
    useOnboardingPlanTransition({ saving: false, finish }, { screen: "billing", goToScreen }, "org"),
  );
  expect(result.current.hasBillingStep).toBe(true);
  Object.assign(billing, activePlans["plan with card on file"]);
  rerender();
  expect(result.current.hasBillingStep).toBe(true);
});
it("sends the agent screen back to the plan step for an inactive plan", async () => {
  const finish = vi.fn(async () => {});
  const goToScreen = vi.fn();
  renderHook(() => useOnboardingPlanTransition({ saving: false, finish }, { screen: "agent", goToScreen }, "org"));
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("billing"));
});
