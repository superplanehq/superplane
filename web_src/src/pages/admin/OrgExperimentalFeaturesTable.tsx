import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import {
  useAdminExperimentalFeaturesRegistry,
  useToggleAdminExperimentalFeature,
  useToggleAdminExperimentalFeatures,
} from "@/hooks/useAdminExperimentalFeatures";
import { Switch } from "@/ui/switch";
import { FlaskConical } from "lucide-react";
import { useMemo, useState } from "react";

export function OrgExperimentalFeaturesTable({ orgId }: { orgId: string }) {
  const { data: registry, isLoading } = useAdminExperimentalFeaturesRegistry(orgId);
  const toggleFeature = useToggleAdminExperimentalFeature(orgId);
  const toggleFeatures = useToggleAdminExperimentalFeatures(orgId);
  const [error, setError] = useState<string | null>(null);

  const features = registry?.features ?? [];
  const enabled = useMemo(() => new Set(registry?.enabled ?? []), [registry?.enabled]);
  const pendingId = toggleFeature.isPending ? (toggleFeature.variables?.featureId ?? null) : null;
  const isBulkBusy = toggleFeatures.isPending;

  const visible = features.filter((f) => !f.released);
  const allVisibleOn = visible.length > 0 && visible.every((feature) => enabled.has(feature.id));

  const handleToggle = (featureId: string, next: boolean) => {
    setError(null);
    toggleFeature.mutate(
      { featureId, enabled: next },
      {
        onError: () => setError(`Failed to ${next ? "enable" : "disable"} ${featureId}`),
      },
    );
  };

  const handleToggleAll = (next: boolean) => {
    const featureIds = visible.filter((feature) => enabled.has(feature.id) !== next).map((feature) => feature.id);
    if (featureIds.length === 0) return;

    setError(null);
    toggleFeatures.mutate(
      { featureIds, enabled: next },
      {
        onError: () => setError(`Failed to ${next ? "enable" : "disable"} experimental features`),
      },
    );
  };

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <FlaskConical size={16} className="text-gray-600 dark:text-gray-400" />
        <Heading level={2} className="text-gray-800 text-base dark:text-gray-100">
          Experimental Features ({visible.length})
        </Heading>
      </div>

      {isLoading ? (
        <Text className="text-gray-500 text-sm dark:text-gray-400">Loading...</Text>
      ) : visible.length === 0 ? (
        <Text className="text-gray-500 text-sm dark:text-gray-400">
          No experimental features are available right now.
        </Text>
      ) : (
        <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-100 dark:border-gray-700/70">
                <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Feature</th>
                <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Description</th>
                <th className="text-right px-4 py-2.5 text-gray-500 font-medium w-32 dark:text-gray-400">
                  <div className="flex flex-col items-end gap-1.5">
                    Status
                    <Switch
                      checked={allVisibleOn}
                      onCheckedChange={handleToggleAll}
                      disabled={isBulkBusy || toggleFeature.isPending}
                      aria-label="Toggle all experimental features"
                    />
                  </div>
                </th>
              </tr>
            </thead>
            <tbody>
              {visible.map((feature) => {
                const isOn = enabled.has(feature.id);
                const isBusy = isBulkBusy || pendingId === feature.id;
                return (
                  <tr key={feature.id} className="border-b border-slate-50 last:border-0 dark:border-gray-800/70">
                    <td className="px-4 py-2.5 text-gray-800 font-medium dark:text-gray-100">{feature.label}</td>
                    <td className="px-4 py-2.5 text-gray-500 dark:text-gray-400">{feature.description || "—"}</td>
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
