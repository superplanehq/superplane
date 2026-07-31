import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { useToggleAdminExperimentalFeature } from "@/hooks/useAdminExperimentalFeatures";
import { useExperimentalFeaturesRegistry } from "@/hooks/useExperimentalFeatures";
import { useOrganization } from "@/hooks/useOrganizationData";
import { Switch } from "@/ui/switch";
import { FlaskConical } from "lucide-react";
import { useMemo, useState } from "react";

export function OrgExperimentalFeaturesTable({ orgId }: { orgId: string }) {
  const { data: registry, isLoading: registryLoading } = useExperimentalFeaturesRegistry();
  const { data: organization, isLoading: orgLoading } = useOrganization(orgId);
  const toggleFeature = useToggleAdminExperimentalFeature(orgId);
  const [error, setError] = useState<string | null>(null);

  const features = registry?.features ?? [];
  const enabled = useMemo(
    () => new Set(organization?.spec?.enabledExperimentalFeatures ?? []),
    [organization?.spec?.enabledExperimentalFeatures],
  );
  const pendingId = toggleFeature.isPending ? (toggleFeature.variables?.featureId ?? null) : null;
  const isLoading = registryLoading || orgLoading;

  const handleToggle = (featureId: string, next: boolean) => {
    setError(null);
    toggleFeature.mutate(
      { featureId, enabled: next },
      {
        onError: () => setError(`Failed to ${next ? "enable" : "disable"} ${featureId}`),
      },
    );
  };

  const visible = features.filter((f) => !f.released);

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <FlaskConical size={16} className="text-content-secondary" />
        <Heading level={2} className="text-base text-content-primary">
          Experimental Features ({visible.length})
        </Heading>
      </div>

      {isLoading ? (
        <Text className="text-sm text-content-secondary">Loading...</Text>
      ) : visible.length === 0 ? (
        <Text className="text-sm text-content-secondary">No experimental features are available right now.</Text>
      ) : (
        <div className="overflow-hidden rounded-md bg-surface-raised shadow-sm outline outline-edge-subtle">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-edge-default">
                <th className="px-4 py-2.5 text-left font-medium text-content-secondary">Feature</th>
                <th className="px-4 py-2.5 text-left font-medium text-content-secondary">Description</th>
                <th className="w-32 px-4 py-2.5 text-right font-medium text-content-secondary">Status</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((feature) => {
                const isOn = enabled.has(feature.id);
                const isBusy = pendingId === feature.id;
                return (
                  <tr key={feature.id} className="border-b border-edge-subtle last:border-0">
                    <td className="px-4 py-2.5 font-medium text-content-primary">{feature.label}</td>
                    <td className="px-4 py-2.5 text-content-secondary">{feature.description || "—"}</td>
                    <td className="px-4 py-2.5 text-right">
                      <Switch
                        checked={isOn}
                        onCheckedChange={(next) => handleToggle(feature.id, next)}
                        disabled={isBusy}
                        aria-label={`Toggle ${feature.label}`}
                      />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {error ? <Text className="text-red-600 text-sm mt-2 dark:text-red-400">{error}</Text> : null}
    </div>
  );
}
