import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { factoriesPreviewFactorySchedule } from "@/api-client";
import { useScheduledIntakeSchedule } from "./useScheduledIntakeSchedule";

vi.mock("@/api-client", () => ({ factoriesPreviewFactorySchedule: vi.fn() }));

const response = (nextRun: string) => ({
  data: { nextRun },
  request: new Request("http://localhost/preview"),
  response: new Response(),
});
type PreviewResponse = ReturnType<typeof response>;
const params = { type: "days", hour: "09", minute: "00", timezone: "0" };

function setup(cached?: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 300_000 } } });
  if (cached) client.setQueryData(["factory-schedule-preview", "org", "factory", params], cached);
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return { client, ...renderHook(() => useScheduledIntakeSchedule("org", "factory", true), { wrapper }) };
}

async function advance(ms: number) {
  await act(async () => {});
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.spyOn(Date.prototype, "getTimezoneOffset").mockReturnValue(0);
  vi.mocked(factoriesPreviewFactorySchedule).mockReset();
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("scheduled intake preview expiration", () => {
  it("blocks creation and refreshes when the next run expires", async () => {
    const nextRun = new Date(Date.now() + 10_000).toISOString();
    let finishRefresh!: (value: PreviewResponse) => void;
    vi.mocked(factoriesPreviewFactorySchedule)
      .mockResolvedValueOnce(response(nextRun))
      .mockImplementationOnce(
        () =>
          new Promise<PreviewResponse>((resolve) => {
            finishRefresh = resolve;
          }),
      );
    const { result, client } = setup();
    await advance(1);
    expect(result.current.ready).toBe(true);
    await advance(10_000);
    expect(result.current.ready).toBe(false);
    expect(result.current.checking).toBe(true);
    expect(factoriesPreviewFactorySchedule).toHaveBeenCalledTimes(2);
    const updated = new Date(Date.now() + 10_000).toISOString();
    await act(async () => {
      finishRefresh(response(updated));
    });
    await advance(1);
    expect(result.current.ready).toBe(true);
    expect(result.current.nextRun).toBe(updated);
    client.clear();
  });

  it("refreshes expired cached data when reopened despite the global stale time", async () => {
    let finishRefresh!: (value: PreviewResponse) => void;
    vi.mocked(factoriesPreviewFactorySchedule).mockImplementationOnce(
      () =>
        new Promise<PreviewResponse>((resolve) => {
          finishRefresh = resolve;
        }),
    );
    const { result, client } = setup(new Date(Date.now() - 1_000).toISOString());
    expect(result.current.ready).toBe(false);
    expect(result.current.checking).toBe(true);
    await advance(1);
    expect(factoriesPreviewFactorySchedule).toHaveBeenCalledTimes(1);
    const nextRun = new Date(Date.now() + 10_000).toISOString();
    await act(async () => {
      finishRefresh(response(nextRun));
    });
    await advance(1);
    expect(result.current.ready).toBe(true);
    expect(result.current.nextRun).toBe(nextRun);
    client.clear();
  });
});
