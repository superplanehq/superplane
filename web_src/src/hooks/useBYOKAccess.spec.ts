import { renderHook } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "bun:test";
import { useBYOKAccess } from "./useBYOKAccess";

let flags: string[];
let productFeatures: Record<string, boolean> | undefined;
let failed: boolean;
let pending: boolean;
let registryFailed: boolean;
let organizationReady: boolean;
const refetch = vi.fn(async () => ({ data: { productFeatures }, error: failed ? new Error("unavailable") : null }));
vi.mock("./useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    organizationReady,
    lookupFailed: registryFailed,
    isLoading: false,
    has: (key: string) => flags.includes(key),
  }),
}));
vi.mock("./useOrganizationBilling", () => ({
  useOrganizationBilling: () => ({
    data: { productFeatures },
    isSuccess: !failed && !pending,
    isError: failed,
    isPending: pending,
    error: failed ? new Error("unavailable") : null,
    refetch,
  }),
}));
beforeEach(() => {
  flags = [];
  productFeatures = { byo_key: false };
  failed = false;
  pending = false;
  registryFailed = false;
  organizationReady = true;
});

it.each([
  [false, false, false],
  [true, false, false],
  [false, true, false],
  [true, true, true],
])("requires rollout %s and product permission %s", (rollout, permission, allowed) => {
  flags = rollout ? ["organization_byok"] : [];
  productFeatures = { byo_key: permission };
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.allowed).toBe(allowed);
  expect(result.current.customProviderAllowed).toBe(false);
});

it("requires a separate custom-provider rollout", () => {
  flags = ["organization_byok", "organization_byok_custom_provider"];
  productFeatures = { byo_key: true };
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.customProviderAllowed).toBe(true);
});

it.each(["pending", "failed", "missing", "registry", "organization"])(
  "does not confirm access when lookup is %s",
  async (state) => {
    flags = ["organization_byok"];
    pending = state === "pending";
    failed = state === "failed";
    registryFailed = state === "registry";
    organizationReady = state !== "organization";
    productFeatures = state === "missing" ? undefined : { byo_key: true };
    const { result } = renderHook(() => useBYOKAccess("org"));
    expect(result.current.ready).toBe(false);
    expect(result.current.allowed).toBe(false);
    if (!pending) await expect(result.current.verify()).rejects.toThrow("could not verify");
  },
);

it("branches on the refreshed permission", async () => {
  flags = ["organization_byok"];
  productFeatures = { byo_key: false };
  const { result } = renderHook(() => useBYOKAccess("org"));
  productFeatures = { byo_key: true };
  expect(await result.current.verify()).toBe(true);
});
