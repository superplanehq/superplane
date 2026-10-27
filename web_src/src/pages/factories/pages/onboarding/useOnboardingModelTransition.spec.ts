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
  Object.assign(cardStep, { ready: true, available: false, required: false, plans: [], planActive: false });
});
function mount(saving = false, screen: "tickets" | "card" = "tickets") {
  const finish = vi.fn(async () => {});
  const goToScreen = vi.fn();
  const hook = renderHook(
    ({ saving }) => useOnboardingModelTransition({ saving, finish }, { screen, goToScreen }, "org"),
    { initialProps: { saving } },
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
  rerender({ saving: false });
  await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
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
  const { result, goToScreen } = mount(false, "card");
  expect(result.current.screen).toBe("complete");
  await waitFor(() => expect(goToScreen).toHaveBeenCalledWith("complete"));
});
