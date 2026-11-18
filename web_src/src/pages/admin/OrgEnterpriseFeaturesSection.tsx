import {
  enterpriseFeaturesListOrganizationEnterpriseFeatures,
  enterpriseFeaturesSetOrganizationEnterpriseFeatures,
} from "@/api-client/sdk.gen";
import type { EnterpriseFeaturesOrganizationEnterpriseFeature } from "@/api-client/types.gen";
import { Text } from "@/components/Text/text";
import { enterpriseFeatureLabel } from "@/lib/license";
import { Switch } from "@/ui/switch";
import { useEffect, useState } from "react";
import { Link } from "react-router";
import { formatDate } from "./formatDate";

type FeatureRow = EnterpriseFeaturesOrganizationEnterpriseFeature;

const apiMessage = (error: unknown, fallback: string) => {
  if (error && typeof error === "object" && "message" in error && typeof error.message === "string" && error.message) {
    return error.message;
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
};

const grantDetail = (feature: FeatureRow) => {
  if (!feature.granted) {
    return null;
  }
  if (feature.source === "auto_single_org") {
    return "Enabled automatically";
  }
  if (feature.source !== "admin") {
    return null;
  }

  const name = feature.grantedByName || "an administrator";
  if (!feature.createdAt) {
    return `Enabled by ${name}`;
  }
  return `Enabled by ${name} on ${formatDate(feature.createdAt)}`;
};

export function OrgEnterpriseFeaturesSection({ orgId }: { orgId: string }) {
  const [features, setFeatures] = useState<FeatureRow[]>([]);
  const [licenseActive, setLicenseActive] = useState(true);
  const [loading, setLoading] = useState(true);
  const [pendingKey, setPendingKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError(null);
    enterpriseFeaturesListOrganizationEnterpriseFeatures({ path: { organizationId: orgId } })
      .then((response) => {
        if (!active) return;
        setFeatures(response.data?.features ?? []);
        setLicenseActive(response.data?.licenseActive ?? false);
      })
      .catch((loadError: unknown) => {
        if (active) setError(apiMessage(loadError, "Failed to load Enterprise features."));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [orgId]);

  const save = async (key: string, enabled: boolean) => {
    const current = features
      .filter((feature) => feature.granted && feature.key)
      .map((feature) => feature.key as string);
    const next = enabled ? [...current, key] : current.filter((item) => item !== key);
    setPendingKey(key);
    setError(null);
    try {
      const response = await enterpriseFeaturesSetOrganizationEnterpriseFeatures({
        path: { organizationId: orgId },
        body: { features: next },
      });
      setFeatures(response.data?.features ?? []);
    } catch (saveError: unknown) {
      setError(apiMessage(saveError, "Failed to save Enterprise features."));
    } finally {
      setPendingKey(null);
    }
  };

  return (
    <section className="mb-8" data-testid="org-enterprise-features">
      <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Enterprise features</h2>
      {licenseActive ? null : (
        <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          No active Enterprise license.{" "}
          <Link to="/admin/license" className="font-medium underline underline-offset-2">
            License
          </Link>
        </Text>
      )}

      {loading ? (
        <Text className="mt-3 text-sm text-gray-500 dark:text-gray-400">Loading Enterprise features...</Text>
      ) : null}

      {!loading && features.length === 0 ? (
        <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">
          This organization has no Enterprise features.
        </Text>
      ) : null}

      {!loading && features.length > 0 ? (
        <div className="mt-3 overflow-hidden rounded-lg border border-slate-200 dark:border-gray-700">
          <table className="w-full text-sm">
            <tbody>
              {features.map((feature) => {
                const key = feature.key ?? "";
                const label = enterpriseFeatureLabel(key);
                const canEnable = licenseActive && feature.licensed === true;
                const detail = grantDetail(feature);
                return (
                  <tr key={key} className="border-b border-slate-50 last:border-0 dark:border-gray-800/70">
                    <td className="px-4 py-2.5">
                      <p className="font-medium text-gray-900 dark:text-gray-100">{label}</p>
                      {feature.licensed === false && feature.granted ? (
                        <Text className="text-xs text-gray-500 dark:text-gray-400">Not in the current license.</Text>
                      ) : null}
                      {detail ? <Text className="text-xs text-gray-500 dark:text-gray-400">{detail}</Text> : null}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <Switch
                        checked={feature.granted === true}
                        onCheckedChange={(next) => void save(key, next)}
                        disabled={pendingKey !== null || (feature.granted !== true && !canEnable)}
                        aria-label={`Toggle ${label}`}
                      />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}

      {error ? (
        <Text className="mt-2 text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </Text>
      ) : null}
    </section>
  );
}
