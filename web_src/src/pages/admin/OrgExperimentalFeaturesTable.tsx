import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { Switch } from "@/ui/switch";
import { useQueryClient } from "@tanstack/react-query";
import { FlaskConical } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

interface ExperimentalFeature {
  id: string;
  label: string;
  description: string;
  released: boolean;
}

interface RegistryResponse {
  features: ExperimentalFeature[];
  enabled: string[];
}

export function OrgExperimentalFeaturesTable({ orgId }: { orgId: string }) {
  const queryClient = useQueryClient();
  const [features, setFeatures] = useState<ExperimentalFeature[]>([]);
  const [enabled, setEnabled] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchFeatures = useCallback(async () => {
    setLoading(true);
    const res = await fetch(`/admin/api/organizations/${orgId}/experimental-features`, {
      credentials: "include",
    });
    if (res.ok) {
      const data: RegistryResponse = await res.json();
      setFeatures(data.features ?? []);
      setEnabled(new Set(data.enabled ?? []));
    }
    setLoading(false);
  }, [orgId]);

  useEffect(() => {
    fetchFeatures();
  }, [fetchFeatures]);

  const handleToggle = async (featureId: string, next: boolean) => {
    setPendingId(featureId);
    setError(null);

    const previous = new Set(enabled);
    const optimistic = new Set(previous);
    if (next) optimistic.add(featureId);
    else optimistic.delete(featureId);
    setEnabled(optimistic);

    const res = await fetch(`/admin/api/organizations/${orgId}/experimental-features/${featureId}`, {
      method: next ? "POST" : "DELETE",
      credentials: "include",
    });

    if (!res.ok) {
      setEnabled(previous);
      setError(`Failed to ${next ? "enable" : "disable"} ${featureId}`);
    } else {
      queryClient.invalidateQueries({ queryKey: organizationKeys.details(orgId) });
    }

    setPendingId(null);
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

      {loading ? (
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
