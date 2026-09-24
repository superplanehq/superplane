import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { meDescribeLastLocation, meSaveLastLocation } = vi.hoisted(() => ({
  meDescribeLastLocation: vi.fn(),
  meSaveLastLocation: vi.fn(),
}));

vi.mock("@/api-client", () => ({ meDescribeLastLocation, meSaveLastLocation }));

import { fetchLastLocationPath, saveLastLocation, useLastLocation } from "./useLastLocation";

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useLastLocation", () => {
  beforeEach(() => {
    meDescribeLastLocation.mockReset();
    meSaveLastLocation.mockReset();
  });

  it("is disabled without an organization route", () => {
    const { result } = renderHook(() => useLastLocation(null), { wrapper: createWrapper() });
    expect(result.current.isPending).toBe(true);
    expect(meDescribeLastLocation).not.toHaveBeenCalled();
  });

  it("returns the saved path", async () => {
    meDescribeLastLocation.mockResolvedValue({ data: { lastLocation: { path: "/acme/apps/deploy?run=1" } } });

    const { result } = renderHook(() => useLastLocation("acme"), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBe("/acme/apps/deploy?run=1");
  });

  it("returns null when there is nothing saved", async () => {
    meDescribeLastLocation.mockResolvedValue({ data: {} });

    const { result } = renderHook(() => useLastLocation("acme"), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBeNull();
  });

  it("treats an unsafe saved path as absent", async () => {
    meDescribeLastLocation.mockResolvedValue({ data: { lastLocation: { path: "//evil.com" } } });

    const { result } = renderHook(() => useLastLocation("acme"), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBeNull();
  });
});

describe("fetchLastLocationPath", () => {
  beforeEach(() => {
    meDescribeLastLocation.mockReset();
  });

  it("returns the saved path", async () => {
    meDescribeLastLocation.mockResolvedValue({ data: { lastLocation: { path: "/acme/apps/deploy?run=1" } } });
    await expect(fetchLastLocationPath("acme")).resolves.toBe("/acme/apps/deploy?run=1");
  });

  it("returns null when the request fails", async () => {
    meDescribeLastLocation.mockRejectedValue(new Error("boom"));
    await expect(fetchLastLocationPath("acme")).resolves.toBeNull();
  });

  it("returns null when the saved path belongs to another organization", async () => {
    meDescribeLastLocation.mockResolvedValue({ data: { lastLocation: { path: "/other/apps" } } });
    await expect(fetchLastLocationPath("acme")).resolves.toBeNull();
  });
});

describe("saveLastLocation", () => {
  beforeEach(() => {
    meSaveLastLocation.mockReset();
  });

  it("saves a safe path", async () => {
    meSaveLastLocation.mockResolvedValue({ data: {} });

    await saveLastLocation("acme", "/acme/apps/deploy?run=1");

    expect(meSaveLastLocation).toHaveBeenCalledWith(
      expect.objectContaining({ body: { path: "/acme/apps/deploy?run=1" } }),
    );
  });

  it("does not call the backend for an unsafe path", async () => {
    await saveLastLocation("acme", "//evil.com");
    expect(meSaveLastLocation).not.toHaveBeenCalled();
  });

  it("does not call the backend for another organization's path", async () => {
    await saveLastLocation("acme", "/other/apps");
    expect(meSaveLastLocation).not.toHaveBeenCalled();
  });

  it("swallows backend errors", async () => {
    meSaveLastLocation.mockRejectedValue(new Error("boom"));
    await expect(saveLastLocation("acme", "/acme")).resolves.toBeUndefined();
  });
});
