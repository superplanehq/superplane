import { enterpriseFeaturesListLicenseActivations } from "@/api-client/sdk.gen";
import type { EnterpriseFeaturesLicenseActivation } from "@/api-client/types.gen";
import { Text } from "@/components/Text/text";
import { enterpriseFeatureLabel } from "@/lib/license";
import { useEffect, useState } from "react";
import { formatDateTime } from "./formatDate";

const reasonLabels: Record<string, string> = {
  expired: "Expired",
  revoked: "Revoked",
  removed: "Removed",
  replaced: "Replaced",
};

const shortLicenseId = (id?: string) => (id ? id.slice(0, 8) : "—");

export function LicenseActivationHistory() {
  const [activations, setActivations] = useState<EnterpriseFeaturesLicenseActivation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    enterpriseFeaturesListLicenseActivations({ query: { pageSize: 50 } })
      .then((response) => {
        if (active) setActivations(response.data?.activations ?? []);
      })
      .catch(() => {
        if (active) setError("Failed to load license history.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <section className="border-t border-slate-200 py-6 dark:border-gray-700/70" data-testid="license-activations">
      <p className="text-xs font-semibold tracking-wide text-gray-500 uppercase dark:text-gray-400">History</p>
      {loading ? <Text className="mt-3 text-sm text-gray-500">Loading license history...</Text> : null}
      {error ? <Text className="mt-3 text-sm text-red-600 dark:text-red-400">{error}</Text> : null}
      {!loading && !error && activations.length === 0 ? (
        <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">No license activations.</Text>
      ) : null}
      {!loading && activations.length > 0 ? (
        <div className="mt-3 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="text-xs text-gray-500 dark:text-gray-400">
                <th className="py-2 pr-3 font-medium">Activated</th>
                <th className="py-2 pr-3 font-medium">License</th>
                <th className="py-2 pr-3 font-medium">Features</th>
                <th className="py-2 pr-3 font-medium">Organizations</th>
                <th className="py-2 pr-3 font-medium">Ended</th>
                <th className="py-2 pr-3 font-medium">Reason</th>
                <th className="py-2 font-medium">Removed grants</th>
              </tr>
            </thead>
            <tbody>
              {activations.map((activation) => {
                const removed = activation.removedGrants ?? [];
                return (
                  <tr key={activation.id} className="border-t border-slate-100 align-top dark:border-gray-800">
                    <td className="py-2 pr-3">{formatDateTime(activation.activatedAt)}</td>
                    <td className="py-2 pr-3 font-mono text-xs">{shortLicenseId(activation.licenseId)}</td>
                    <td className="py-2 pr-3">
                      {(activation.licenseFeatures ?? []).map(enterpriseFeatureLabel).join(", ") || "—"}
                    </td>
                    <td className="py-2 pr-3">{activation.organizationCount ?? 0}</td>
                    <td className="py-2 pr-3">
                      {activation.deactivatedAt ? formatDateTime(activation.deactivatedAt) : "—"}
                    </td>
                    <td className="py-2 pr-3">
                      {activation.deactivationReason
                        ? (reasonLabels[activation.deactivationReason] ?? activation.deactivationReason)
                        : "—"}
                    </td>
                    <td className="py-2">
                      {removed.length === 0 ? (
                        "0"
                      ) : (
                        <details>
                          <summary>
                            {removed.length} removed {removed.length === 1 ? "grant" : "grants"}
                          </summary>
                          <ul className="mt-1 space-y-1 text-xs text-gray-600 dark:text-gray-400">
                            {removed.map((grant, index) => (
                              <li key={`${grant.organizationId}-${grant.featureKey}-${index}`}>
                                {grant.organizationName || "Deleted organization"} ·{" "}
                                {enterpriseFeatureLabel(grant.featureKey ?? "")}
                              </li>
                            ))}
                          </ul>
                        </details>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
