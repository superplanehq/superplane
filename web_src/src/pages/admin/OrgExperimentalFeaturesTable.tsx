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
        <FlaskConical size={16} className="text-gray-600" />
        <Heading level={2} className="text-gray-800 text-base">
          Experimental Features ({visible.length})
        </Heading>
      </div>

      {isLoading ? (
        <Text className="text-gray-500 text-sm">Loading...</Text>
      ) : visible.length === 0 ? (
        <Text className="text-gray-500 text-sm">No experimental features are available right now.</Text>
      ) : (
        <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-100">
                <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Feature</th>
                <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Description</th>
                <th className="text-right px-4 py-2.5 text-gray-500 font-medium w-32">Status</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((feature) => {
                const isOn = enabled.has(feature.id);
                const isBusy = pendingId === feature.id;
                return (
                  <tr key={feature.id} className="border-b border-slate-50 last:border-0">
                    <td className="px-4 py-2.5 text-gray-800 font-medium">{feature.label}</td>
                    <td className="px-4 py-2.5 text-gray-500">{feature.description || "—"}</td>
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

      {error ? <Text className="text-red-600 text-sm mt-2">{error}</Text> : null}
    </div>
  );
}
