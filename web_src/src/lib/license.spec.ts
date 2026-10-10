import { afterEach, describe, expect, it, vi } from "bun:test";

import {
  daysUntil,
  fetchInstallationLicenseWhenKeysReady,
  licenseExpiryWarning,
  licenseReasonMessage,
  type TrustedKeysState,
} from "./license";

const now = new Date("2026-10-05T12:00:00Z");
const inDays = (days: number) => new Date(now.getTime() + days * 24 * 60 * 60 * 1000).toISOString();

describe("licenseExpiryWarning", () => {
  it("returns null without license details", () => {
    expect(licenseExpiryWarning(undefined, now)).toBeNull();
    expect(licenseExpiryWarning({ edition: "community", features: [] }, now)).toBeNull();
  });

  it("returns null when the active license expires after the warning window", () => {
    const license = { edition: "enterprise" as const, features: [], state: "active" as const, expires_at: inDays(31) };
    expect(licenseExpiryWarning(license, now)).toBeNull();
  });

  it("warns when the active license expires within the warning window", () => {
    const license = { edition: "enterprise" as const, features: [], state: "active" as const, expires_at: inDays(10) };
    expect(licenseExpiryWarning(license, now)).toEqual({ expired: false, daysLeft: 10 });
  });

  it("warns when the license has expired", () => {
    const license = { edition: "community" as const, features: [], state: "expired" as const, expires_at: inDays(-2) };
    expect(licenseExpiryWarning(license, now)).toEqual({ expired: true, daysLeft: 0 });
  });

  it("does not warn for a revoked license", () => {
    const license = {
      edition: "community" as const,
      features: [],
      state: "revoked" as const,
      expires_at: inDays(5),
    };
    expect(licenseExpiryWarning(license, now)).toBeNull();
  });

  it("does not warn for a license that is not valid yet", () => {
    const license = {
      edition: "community" as const,
      features: [],
      state: "not_yet_valid" as const,
      expires_at: inDays(5),
    };
    expect(licenseExpiryWarning(license, now)).toBeNull();
  });
});

describe("daysUntil", () => {
  it("rounds partial days up", () => {
    expect(daysUntil(new Date(now.getTime() + 60 * 60 * 1000).toISOString(), now)).toBe(1);
  });
});

describe("licenseReasonMessage", () => {
  it("maps known reasons and falls back for unknown ones", () => {
    expect(licenseReasonMessage("expired")).toBe("The license has expired.");
    expect(licenseReasonMessage("something_else")).toBe("The license could not be read.");
    expect(licenseReasonMessage(undefined)).toBe("The license could not be read.");
  });
});

describe("fetchInstallationLicenseWhenKeysReady", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  const statusWithKeys = (state: TrustedKeysState) =>
    new Response(
      JSON.stringify({
        edition: "community",
        state: "none",
        source: "database",
        managed_by_configuration: false,
        trusted_keys: { state, version: 1 },
      }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    );

  it("waits until the first key download finishes", async () => {
    const states: TrustedKeysState[] = ["syncing", "syncing", "synced"];
    const fetchMock = vi.fn(async () => statusWithKeys(states.shift() ?? "synced"));
    vi.stubGlobal("fetch", fetchMock);

    const status = await fetchInstallationLicenseWhenKeysReady(1_000, 1);

    expect(status.trusted_keys?.state).toBe("synced");
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("stops waiting after the time limit", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => statusWithKeys("syncing")),
    );

    const status = await fetchInstallationLicenseWhenKeysReady(20, 5);

    expect(status.trusted_keys?.state).toBe("syncing");
  });
});
