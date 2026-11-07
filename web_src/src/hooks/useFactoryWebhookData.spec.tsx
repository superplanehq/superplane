import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "bun:test";
import type { ReactNode } from "react";

import { useFactoryWebhookData } from "./useFactoryWebhookData";

const failed = {
  id: "hook-1",
  repository: "acme/app",
  eventTypes: ["issues"],
  state: "failed",
  message: "GitHub could not register the webhook for issues. Check the repository settings, then try again.",
};
let remoteWebhook = failed;
const listWebhooks = vi.fn(async () => ({ data: { webhooks: [{ ...remoteWebhook }] } }));
const retryWebhook = vi.fn(async () => {
  remoteWebhook = { ...failed, state: "pending" };
  return { data: { webhook: remoteWebhook } };
});

vi.mock("@/api-client", () => ({
  factoriesListFactoryWebhooks: listWebhooks,
  factoriesRetryFactoryWebhook: retryWebhook,
}));

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn() }));

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

afterEach(() => {
  vi.useRealTimers();
  listWebhooks.mockClear();
  retryWebhook.mockClear();
  remoteWebhook = failed;
});

describe("useFactoryWebhookData", () => {
  it("keeps a warning during retry, restores retry after another failure, and clears it on success", async () => {
    const { result } = renderHook(() => useFactoryWebhookData("org-1", "factory-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.warnings).toHaveLength(1));

    await act(async () => result.current.retry(failed));
    await waitFor(() => expect(result.current.retrying(failed.id)).toBe(true));
    expect(result.current.warnings).toHaveLength(1);

    remoteWebhook = { ...failed };
    await act(async () => result.current.refresh());
    await waitFor(() => expect(result.current.retrying(failed.id)).toBe(false));
    expect(result.current.warnings).toHaveLength(1);

    await act(async () => result.current.retry(failed));
    remoteWebhook = { ...failed, state: "ready" };
    await act(async () => result.current.refresh());
    await waitFor(() => expect(result.current.warnings).toHaveLength(0));
  });

  it("stops polling after one minute and allows a manual refresh", async () => {
    const { result } = renderHook(() => useFactoryWebhookData("org-1", "factory-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.warnings).toHaveLength(1));
    vi.useFakeTimers();
    await act(async () => result.current.retry(failed));
    expect(result.current.retrying(failed.id)).toBe(true);

    await act(async () => vi.advanceTimersByTimeAsync(60_000));
    expect(result.current.retryTimedOut(failed.id)).toBe(true);
    const callsAtTimeout = listWebhooks.mock.calls.length;
    await act(async () => vi.advanceTimersByTimeAsync(4000));
    expect(listWebhooks).toHaveBeenCalledTimes(callsAtTimeout);

    await act(async () => result.current.refresh());
    expect(listWebhooks).toHaveBeenCalledTimes(callsAtTimeout + 1);
  });
});
