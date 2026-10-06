export type LicenseEdition = "community" | "enterprise";
export type LicenseState = "none" | "active" | "expired" | "not_yet_valid" | "invalid";
export type LicenseSource = "none" | "file" | "database";
export type TrustedKeysState = "disabled" | "syncing" | "synced" | "failed";

export type TrustedKeys = {
  state: TrustedKeysState;
  version: number;
  synced_at?: string;
};

export type InstallationLicense = {
  edition: LicenseEdition;
  state: LicenseState;
  source: LicenseSource;
  reason?: string;
  managed_by_configuration: boolean;
  license?: {
    id: string;
    customer_id: string;
    features: string[];
    issued_at: string;
    valid_from: string;
    expires_at: string;
  };
  trusted_keys?: TrustedKeys;
};

export type AccountLicense = {
  edition: LicenseEdition;
  features: string[];
  state?: LicenseState;
  expires_at?: string;
};

export const ENTERPRISE_FEATURES = [
  { key: "custom_roles", label: "Custom roles", description: "Define roles with specific permissions." },
  { key: "groups", label: "Groups", description: "Manage access for teams of members." },
] as const;

export const LICENSE_EXPIRY_WARNING_DAYS = 30;

const LICENSE_PATH = "/admin/api/installation/license";
const KEY_LIST_PATH = `${LICENSE_PATH}/keys`;
const KEY_SYNC_WAIT_MS = 15_000;
const KEY_SYNC_POLL_MS = 1_000;
const DAY_MS = 24 * 60 * 60 * 1000;

const reasonMessages: Record<string, string> = {
  malformed: "The license is not in a valid format.",
  unsupported_algorithm: "The license signature is not valid.",
  invalid_signature: "The license signature is not valid.",
  unknown_key:
    "SuperPlane does not trust the key that signed the license. Upload the latest key list, or check that SuperPlane can download license key updates.",
  invalid_claims: "The license is not valid for self-hosted SuperPlane.",
  expired: "The license has expired.",
  not_yet_valid: "The license is not valid yet.",
  unreadable: "The license could not be read.",
};

export const licenseReasonMessage = (reason?: string) =>
  reasonMessages[reason ?? ""] ?? "The license could not be read.";

export const daysUntil = (isoDate: string, now: Date = new Date()) =>
  Math.ceil((new Date(isoDate).getTime() - now.getTime()) / DAY_MS);

// Returns the number of days left when an installation admin should renew the
// license, or null when no warning is needed.
export const licenseExpiryWarning = (
  license: AccountLicense | undefined,
  now: Date = new Date(),
): { expired: boolean; daysLeft: number } | null => {
  if (!license?.expires_at) {
    return null;
  }

  if (license.state === "expired") {
    return { expired: true, daysLeft: 0 };
  }

  if (license.state !== "active") {
    return null;
  }

  const daysLeft = daysUntil(license.expires_at, now);
  if (daysLeft > LICENSE_EXPIRY_WARNING_DAYS) {
    return null;
  }

  return { expired: false, daysLeft: Math.max(daysLeft, 0) };
};

const responseError = async (response: Response, fallback: string) => {
  const text = (await response.text()).trim();
  return new Error(text === "" ? fallback : text);
};

const parseLicenseResponse = async (response: Response, fallback: string): Promise<InstallationLicense> => {
  if (!response.ok) {
    throw await responseError(response, fallback);
  }

  return response.json();
};

export const fetchInstallationLicense = async () =>
  parseLicenseResponse(await fetch(LICENSE_PATH, { credentials: "include" }), "Failed to load the license");

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

// Waits up to 15 seconds while the server downloads the license keys for the
// first time, so that a new license is not rejected only because the keys are
// not ready yet. Then it returns the status as it is.
export const fetchInstallationLicenseWhenKeysReady = async (
  waitMs: number = KEY_SYNC_WAIT_MS,
  pollMs: number = KEY_SYNC_POLL_MS,
) => {
  const deadline = Date.now() + waitMs;
  let status = await fetchInstallationLicense();
  while (status.trusted_keys?.state === "syncing" && Date.now() < deadline) {
    await delay(pollMs);
    status = await fetchInstallationLicense();
  }

  return status;
};

export const uploadLicenseKeyList = async (keyList: string) =>
  parseLicenseResponse(
    await fetch(KEY_LIST_PATH, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ key_list: keyList.trim() }),
    }),
    "Failed to upload the key list",
  );

export const installInstallationLicense = async (license: string) =>
  parseLicenseResponse(
    await fetch(LICENSE_PATH, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ license: license.trim() }),
    }),
    "Failed to install the license",
  );

export const removeInstallationLicense = async () =>
  parseLicenseResponse(
    await fetch(LICENSE_PATH, { method: "DELETE", credentials: "include" }),
    "Failed to remove the license",
  );
