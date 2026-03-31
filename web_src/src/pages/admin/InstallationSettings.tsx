import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import React, { useCallback, useEffect, useState } from "react";

type InstallationNetworkSettings = {
  allow_private_network_access: boolean;
  effective_blocked_http_hosts: string[];
  effective_private_ip_ranges: string[];
  blocked_http_hosts_overridden: boolean;
  private_ip_ranges_overridden: boolean;
};

const InstallationSettings: React.FC = () => {
  const [settings, setSettings] = useState<InstallationNetworkSettings | null>(null);
  const [allowPrivateNetworkAccess, setAllowPrivateNetworkAccess] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const loadSettings = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch("/admin/api/installation/network-settings", { credentials: "include" });
      if (!response.ok) {
        throw new Error();
      }

      const data: InstallationNetworkSettings = await response.json();
      setSettings(data);
      setAllowPrivateNetworkAccess(data.allow_private_network_access);
    } catch {
      showErrorToast("Failed to load installation settings");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  const saveSettings = async () => {
    setSaving(true);
    try {
      const response = await fetch("/admin/api/installation/network-settings", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          allow_private_network_access: allowPrivateNetworkAccess,
        }),
      });

      if (!response.ok) {
        throw new Error();
      }

      const data: InstallationNetworkSettings = await response.json();
      setSettings(data);
      setAllowPrivateNetworkAccess(data.allow_private_network_access);
      showSuccessToast("Installation settings updated");
    } catch {
      showErrorToast("Failed to update installation settings");
    } finally {
      setSaving(false);
    }
  };

  if (loading && !settings) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
        <Text className="text-gray-500">Loading installation settings...</Text>
      </div>
    );
  }

  const blockedHosts = settings?.effective_blocked_http_hosts ?? [];
  const privateRanges = settings?.effective_private_ip_ranges ?? [];
  const hasChanges = settings != null && allowPrivateNetworkAccess !== settings.allow_private_network_access;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-gray-900">Installation Settings</h1>
        <Text className="mt-1 text-sm text-gray-500">
          Configure network access for integrations, components, and triggers that issue outbound HTTP requests.
        </Text>
      </div>

      <div className="rounded-md bg-white p-6 shadow-sm outline outline-slate-950/10">
        <div className="flex items-start justify-between gap-6">
          <div className="max-w-2xl">
            <h2 className="text-base font-medium text-gray-900">Private network access</h2>
            <Text className="mt-2 text-sm text-gray-600">
              When enabled, SuperPlane can connect to tools behind private IP ranges or internal Kubernetes DNS names.
              Disabling this keeps the default SSRF protections for private networks.
            </Text>
          </div>

          <label className="inline-flex items-center gap-2 text-sm font-medium text-gray-800">
            <input
              type="checkbox"
              checked={allowPrivateNetworkAccess}
              onChange={(event) => setAllowPrivateNetworkAccess(event.target.checked)}
            />
            Allow private network access
          </label>
        </div>

        <div className="mt-6 flex items-center gap-3">
          <Button type="button" onClick={saveSettings} disabled={saving || !hasChanges}>
            {saving ? "Saving..." : "Save settings"}
          </Button>
          {settings?.blocked_http_hosts_overridden || settings?.private_ip_ranges_overridden ? (
            <Text className="text-xs text-amber-700">
              Environment overrides are active. `BLOCKED_HTTP_HOSTS` and `BLOCKED_PRIVATE_IP_RANGES` take precedence
              over this toggle.
            </Text>
          ) : (
            <Text className="text-xs text-gray-500">
              Changes apply without a restart and may take a few seconds to propagate across all app instances.
            </Text>
          )}
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <div className="rounded-md bg-white p-6 shadow-sm outline outline-slate-950/10">
          <h2 className="text-base font-medium text-gray-900">Effective blocked hosts</h2>
          <Text className="mt-1 text-sm text-gray-500">
            {settings?.blocked_http_hosts_overridden
              ? "Overridden by env var."
              : "Resolved from installation settings."}
          </Text>
          <div className="mt-4 rounded-md bg-slate-50 p-4">
            {blockedHosts.length === 0 ? (
              <Text className="text-sm text-gray-500">No blocked hosts.</Text>
            ) : (
              <ul className="space-y-1 font-mono text-xs text-gray-700">
                {blockedHosts.map((host) => (
                  <li key={host}>{host}</li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div className="rounded-md bg-white p-6 shadow-sm outline outline-slate-950/10">
          <h2 className="text-base font-medium text-gray-900">Effective blocked private IP ranges</h2>
          <Text className="mt-1 text-sm text-gray-500">
            {settings?.private_ip_ranges_overridden ? "Overridden by env var." : "Resolved from installation settings."}
          </Text>
          <div className="mt-4 rounded-md bg-slate-50 p-4">
            {privateRanges.length === 0 ? (
              <Text className="text-sm text-gray-500">No blocked private IP ranges.</Text>
            ) : (
              <ul className="space-y-1 font-mono text-xs text-gray-700">
                {privateRanges.map((range) => (
                  <li key={range}>{range}</li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default InstallationSettings;
