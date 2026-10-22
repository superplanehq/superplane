import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { useOnboardingPlanTransition } from "./useOnboardingPlanTransition";

let newBilling = true;
let billing: OrganizationsDescribeOrganizationBillingResponse;
let lookupFailed = false;
const verify = vi.fn(async () => true);
vi.mock("@/hooks/useBYOKAccess", () => ({ useBYOKAccess: () => ({ allowed: true, verify }) }));
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
    useOnboardingPlanTransition({ saving: false, finish }, { screen: "choose", goToScreen }, "org"),
  );
  return { ...hook, finish, goToScreen };
}

it("opens the plan step before model access is resolved", async () => {
  const { result, finish, goToScreen } = mount();
  await act(async () => result.current.afterRepository());
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
it("keeps repository setup open when billing lookup fails", async () => {
  lookupFailed = true;
  const { result, finish, goToScreen } = mount();
  await act(async () => result.current.afterRepository());
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
  await act(async () => result.current.afterRepository());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("agent"));
  expect(goToScreen).not.toHaveBeenCalledWith("billing");
});
