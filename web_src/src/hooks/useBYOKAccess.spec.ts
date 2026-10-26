import { renderHook } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationWorkspaceUsageResponse } from "@/api-client";
import { useBYOKAccess } from "./useBYOKAccess";

let flags: string[];
let billingEnabled: boolean;
let productFeatures: Record<string, boolean> | undefined;
let failed: boolean;
let pending: boolean;
let registryFailed: boolean;
let organizationReady: boolean;
let workspaceUsage: OrganizationsDescribeOrganizationWorkspaceUsageResponse | undefined;
let usageFailed: boolean;
let usagePending: boolean;
const refetch = vi.fn(async () => ({ data: { productFeatures }, error: failed ? new Error("unavailable") : null }));
const refetchUsage = vi.fn(async () => ({
  data: usagePending ? undefined : workspaceUsage,
  error: usageFailed ? new Error("usage unavailable") : null,
}));
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
    data: { billingEnabled, productFeatures },
    isSuccess: !failed && !pending,
    isError: failed,
    isPending: pending,
    error: failed ? new Error("unavailable") : null,
    refetch,
  }),
}));
vi.mock("./useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: usagePending ? undefined : workspaceUsage,
    isSuccess: !usageFailed && !usagePending,
    isError: usageFailed,
    isPending: usagePending,
    error: usageFailed ? new Error("usage unavailable") : null,
    refetch: refetchUsage,
  }),
}));
beforeEach(() => {
  flags = [];
  billingEnabled = true;
  productFeatures = { byo_key: false };
  failed = false;
  pending = false;
  registryFailed = false;
  organizationReady = true;
  workspaceUsage = { defaultHostedProvider: "anthropic", defaultHostedModel: "claude-sonnet-4-6" };
  usageFailed = false;
  usagePending = false;
});

it.each([
  [false, false, false],
  [true, false, false],
  [false, true, false],
  [true, true, true],
])("requires rollout %s and product permission %s with a hosted default", (rollout, permission, allowed) => {
  flags = rollout ? ["organization_byok"] : [];
  productFeatures = { byo_key: permission };
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.allowed).toBe(allowed);
  expect(result.current.customProviderAllowed).toBe(false);
});

it.each([
  [false, false, false, false],
  [true, false, false, false],
  [false, true, false, false],
  [true, true, false, false],
  [false, false, true, false],
  [true, false, true, false],
  [false, true, true, false],
  [true, true, true, true],
])("requires standard rollout %s, custom rollout %s, and permission %s", (standard, custom, permission, allowed) => {
  workspaceUsage = {};
  flags = [standard ? "organization_byok" : "", custom ? "organization_byok_custom_provider" : ""];
  productFeatures = { byo_key: permission };
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.allowed).toBe(true);
  expect(result.current.customProviderAllowed).toBe(allowed);
});

it.each([false, true])("allows standard providers without a hosted default when billing is %s", async (enabled) => {
  billingEnabled = enabled;
  workspaceUsage = {};
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.ready).toBe(true);
  expect(result.current.allowed).toBe(true);
  expect(result.current.customProviderAllowed).toBe(false);
  expect(await result.current.verify()).toBe(true);
});

it.each([
  { defaultHostedProvider: "anthropic" },
  { defaultHostedModel: "claude-sonnet-4-6" },
  { defaultHostedProvider: " ", defaultHostedModel: "claude-sonnet-4-6" },
  { defaultHostedProvider: "anthropic", defaultHostedModel: " " },
])("allows an incomplete hosted default %j", async (incomplete) => {
  workspaceUsage = incomplete;
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.allowed).toBe(true);
  expect(await result.current.verify()).toBe(true);
});

it.each(["pending", "failed", "missing", "registry", "organization"])(
  "does not need the rollout or product lookup when it is %s without a hosted default",
  async (state) => {
    workspaceUsage = {};
    pending = state === "pending";
    failed = state === "failed";
    registryFailed = state === "registry";
    organizationReady = state !== "organization";
    productFeatures = state === "missing" ? undefined : { byo_key: false };
    const { result } = renderHook(() => useBYOKAccess("org"));
    expect(result.current.ready).toBe(true);
    expect(result.current.allowed).toBe(true);
    expect(result.current.customProviderAllowed).toBe(false);
    expect(await result.current.verify()).toBe(true);
  },
);

it.each(["pending", "failed", "missing"])("does not treat a %s hosted-default lookup as absent", async (state) => {
  workspaceUsage = state === "missing" ? undefined : {};
  usagePending = state === "pending";
  usageFailed = state === "failed";
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.ready).toBe(false);
  expect(result.current.allowed).toBe(false);
  await expect(result.current.verify()).rejects.toThrow("could not verify");
});

it.each(["pending", "failed"])("keeps permitted BYOK access when hosted-default lookup is %s", async (state) => {
  workspaceUsage = {};
  usagePending = state === "pending";
  usageFailed = state === "failed";
  flags = ["organization_byok", "organization_byok_custom_provider"];
  productFeatures = { byo_key: true };
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.ready).toBe(true);
  expect(result.current.allowed).toBe(true);
  expect(result.current.customProviderAllowed).toBe(true);
  expect(await result.current.verify()).toBe(true);
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

it("ends the fallback when a hosted default is configured before verification", async () => {
  workspaceUsage = {};
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.allowed).toBe(true);
  workspaceUsage = { defaultHostedProvider: "openai", defaultHostedModel: "gpt-5" };
  expect(await result.current.verify()).toBe(false);
});

it("allows the fallback when a hosted default is removed before verification", async () => {
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.allowed).toBe(false);
  workspaceUsage = {};
  expect(await result.current.verify()).toBe(true);
});

it("does not authorize a cached fallback when refreshing the default fails", async () => {
  workspaceUsage = {};
  const { result } = renderHook(() => useBYOKAccess("org"));
  expect(result.current.allowed).toBe(true);
  usageFailed = true;
  await expect(result.current.verify()).rejects.toThrow("could not verify");
});
