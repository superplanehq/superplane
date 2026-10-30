import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "bun:test";
import { useOnboardingModelTransition } from "./useOnboardingModelTransition";
import type { OnboardingCardStep } from "./useOnboardingCardStep";

const verify = vi.fn(async () => true);
vi.mock("@/hooks/useBYOKAccess", () => ({ useBYOKAccess: () => ({ allowed: true, ready: true, verify }) }));
const toast = vi.fn();
vi.mock("@/lib/toast", () => ({ showErrorToast: toast }));
const cardStep: OnboardingCardStep = {
  ready: true,
  failed: false,
  available: false,
  required: false,
  requiresAdmin: false,
  plans: [],
  planActive: false,
  hasPaymentMethod: false,
  refetch: async () => undefined,
};
vi.mock("./useOnboardingCardStep", () => ({ useOnboardingCardStep: () => ({ ...cardStep }) }));
beforeEach(() => {
  verify.mockReset().mockResolvedValue(true);
  toast.mockClear();
  Object.assign(cardStep, {
    ready: true,
    failed: false,
    available: false,
    required: false,
    plans: [],
    planActive: false,
  });
});
function mount(
  saving = false,
  codexSubscriptionLoading = false,
  screen: "tickets" | "card" | "agent" | "complete" = "tickets",
) {
  const finish = vi.fn(async () => {});
  const goToScreen = vi.fn();
  const hook = renderHook(
    ({ saving, codexSubscriptionLoading, codexSubscriptionOffered }) =>
      useOnboardingModelTransition(
        { saving, finish, codexSubscriptionLoading, codexSubscriptionOffered },
        { screen, goToScreen },
        "org",
      ),
    { initialProps: { saving, codexSubscriptionLoading, codexSubscriptionOffered: false } },
  );
  return { ...hook, finish, goToScreen };
}

it.each([true, false])("continues Trial setup after verifying model access: %s", async (allowed) => {
  verify.mockResolvedValue(allowed);
  const { result, finish, goToScreen } = mount();
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(verify).toHaveBeenCalledTimes(1));
  if (allowed) {
    await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("agent"));
    expect(finish).not.toHaveBeenCalled();
  } else {
    await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
    expect(goToScreen).not.toHaveBeenCalled();
  }
});

it("waits for saved ticket state before completing setup", async () => {
  verify.mockResolvedValue(false);
  const { result, rerender, finish } = mount(true);
  await act(async () => result.current.afterTickets());
  expect(finish).not.toHaveBeenCalled();
  rerender({ saving: false, codexSubscriptionLoading: false, codexSubscriptionOffered: false });
  await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
});

it("waits for ChatGPT eligibility before it continues saved intake", async () => {
  const { result, rerender, finish, goToScreen } = mount(false, true);
  await act(async () => result.current.afterTickets());
  expect(verify).not.toHaveBeenCalled();
  expect(finish).not.toHaveBeenCalled();
  rerender({ saving: false, codexSubscriptionLoading: false, codexSubscriptionOffered: true });
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("agent"));
  expect(finish).not.toHaveBeenCalled();
});

it("keeps setup open when model access cannot be verified", async () => {
  verify.mockRejectedValue(new Error("Could not verify permissions"));
  const { result, finish, goToScreen } = mount();
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(toast).toHaveBeenCalled());
  expect(finish).not.toHaveBeenCalled();
  expect(goToScreen).not.toHaveBeenCalled();
});

it("opens the card step after the backlog step when a starter plan is not active", async () => {
  Object.assign(cardStep, {
    available: true,
    required: true,
    plans: [{ key: "pay_as_you_go", name: "Pay-as-you-go" }],
  });
  const { result, finish, goToScreen } = mount();
  expect(result.current.hasCardStep).toBe(true);
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("card"));
  expect(verify).not.toHaveBeenCalled();
  expect(finish).not.toHaveBeenCalled();
});

it("skips the card step for an organization that already has a paid plan", async () => {
  Object.assign(cardStep, { available: true, required: false, planActive: true });
  const { result, goToScreen } = mount();
  expect(result.current.hasCardStep).toBe(false);
  await act(async () => result.current.afterTickets());
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("agent"));
});

it("sends a restored card URL to the complete screen when the step is not available", async () => {
  const { result, goToScreen } = mount(false, false, "card");
  expect(result.current.screen).toBe("complete");
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("complete"));
});

it.each(["agent", "complete"] as const)("resumes unpaid setup at plan selection instead of %s", async (screen) => {
  Object.assign(cardStep, { available: true, required: true });
  const { result, goToScreen, finish } = mount(false, false, screen);
  expect(result.current.screen).toBe("card");
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("card"));
  expect(finish).not.toHaveBeenCalled();
});

it.each(["agent", "complete"] as const)("stops waiting and shows a retry when billing fails on %s", async (screen) => {
  Object.assign(cardStep, { ready: false, failed: true });
  const { result, goToScreen } = mount(false, false, screen);
  expect(result.current.pending).toBe(false);
  expect(result.current.screen).toBe("complete");
  if (screen === "agent") await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("complete"));
});

it("keeps waiting while the billing lookup runs", () => {
  Object.assign(cardStep, { ready: false, failed: false });
  const { result } = mount(false, false, "complete");
  expect(result.current.pending).toBe(true);
});
