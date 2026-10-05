export type LicenseEdition = "community" | "enterprise";
export type LicenseState = "none" | "active" | "expired" | "not_yet_valid" | "invalid";
export type LicenseSource = "none" | "file" | "database";

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
const DAY_MS = 24 * 60 * 60 * 1000;

const reasonMessages: Record<string, string> = {
  malformed: "The license is not in a valid format.",
  unsupported_algorithm: "The license signature is not valid.",
  invalid_signature: "The license signature is not valid.",
  unknown_key: "This SuperPlane version does not trust the key that signed the license.",
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
