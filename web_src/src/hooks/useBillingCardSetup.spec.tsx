import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

const createSession = vi.fn();
const refetch = vi.fn();
let hasPaymentMethod = false;
vi.mock("@/api-client", () => ({ organizationsCreatePaymentMethodSession: createSession }));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  useOrganizationBilling: () => ({ data: { hasPaymentMethod }, refetch }),
}));

import { useBillingCardSetup } from "./useBillingCardSetup";

function wrapper({ children }: { children: ReactNode }) {
  return createElement(
    QueryClientProvider,
    { client: new QueryClient({ defaultOptions: { mutations: { retry: false } } }) },
    children,
  );
}

function createPopup() {
  const popup = { closed: false, close: vi.fn(), location: { replace: vi.fn() }, opener: window };
  vi.spyOn(window, "open").mockReturnValue(popup as unknown as Window);
  return popup;
}

beforeEach(() => {
  vi.useFakeTimers();
  hasPaymentMethod = false;
  createSession.mockReset().mockResolvedValue({ data: { portalUrl: "https://polar.example/portal?token=test" } });
  refetch.mockReset().mockResolvedValue({ data: { hasPaymentMethod: true } });
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe("useBillingCardSetup", () => {
  it("keeps existing-card updates open until the user closes Polar, then refreshes billing", async () => {
    hasPaymentMethod = true;
    const popup = createPopup();
    const onSaved = vi.fn();
    const { result, unmount } = renderHook(() => useBillingCardSetup("org-1"), { wrapper });
    await act(async () => result.current.open(onSaved));
    await act(async () => vi.advanceTimersByTime(0));
    expect(popup.location.replace).toHaveBeenCalledWith("https://polar.example/portal/settings?token=test");

    await act(async () => vi.advanceTimersByTime(6000));
    expect(result.current.waitingForCard).toBe(true);
    expect(popup.close).not.toHaveBeenCalled();
    expect(refetch).not.toHaveBeenCalled();
    expect(onSaved).not.toHaveBeenCalled();

    popup.closed = true;
    await act(async () => vi.advanceTimersByTime(2000));
    expect(refetch).toHaveBeenCalledTimes(1);
    expect(result.current.waitingForCard).toBe(false);
    expect(onSaved).toHaveBeenCalledTimes(1);
    unmount();
  });

  it("automatically detects the first saved card and completes without requiring popup closure", async () => {
    const popup = createPopup();
    refetch.mockResolvedValueOnce({ data: { hasPaymentMethod: false } });
    let closedBeforeContinue = false;
    const onSaved = vi.fn(() => {
      closedBeforeContinue = popup.close.mock.calls.length > 0;
    });
    const { result, unmount } = renderHook(() => useBillingCardSetup("org-1"), { wrapper });
    await act(async () => result.current.open(onSaved));
    await act(async () => vi.advanceTimersByTime(0));
    expect(result.current.waitingForCard).toBe(true);
    expect(onSaved).not.toHaveBeenCalled();

    await act(async () => vi.advanceTimersByTime(2000));
    expect(refetch).toHaveBeenCalledTimes(2);
    expect(result.current.waitingForCard).toBe(false);
    expect(popup.closed).toBe(false);
    expect(popup.close).toHaveBeenCalled();
    expect(onSaved).toHaveBeenCalledTimes(1);
    expect(closedBeforeContinue).toBe(true);
    unmount();
  });
});
