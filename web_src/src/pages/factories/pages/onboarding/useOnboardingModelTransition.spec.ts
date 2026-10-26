import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "bun:test";
import { useOnboardingModelTransition } from "./useOnboardingModelTransition";

const verify = vi.fn(async () => true);
vi.mock("@/hooks/useBYOKAccess", () => ({ useBYOKAccess: () => ({ allowed: true, ready: true, verify }) }));
const toast = vi.fn();
vi.mock("@/lib/toast", () => ({ showErrorToast: toast }));
beforeEach(() => {
  verify.mockReset().mockResolvedValue(true);
  toast.mockClear();
});
function mount(saving = false, codexSubscriptionLoading = false) {
  const finish = vi.fn(async () => {});
  const goToScreen = vi.fn();
  const hook = renderHook(
    ({ saving, codexSubscriptionLoading, codexSubscriptionOffered }) =>
      useOnboardingModelTransition(
        { saving, finish, codexSubscriptionLoading, codexSubscriptionOffered },
        { screen: "tickets", goToScreen },
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
