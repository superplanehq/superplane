import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { showErrorToast } from "@/utils/toast";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import type { OrganizationsIntegration } from "@/api-client/types.gen";
import { useAccount } from "@/contexts/AccountContext";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

type Organization = {
  id: string;
  name: string;
};

export function SentrySetup() {
  const navigate = useNavigate();
  const { account, loading: accountLoading } = useAccount();
  const searchParams = new URLSearchParams(window.location.search);
  const code = searchParams.get("code") ?? "";
  const installationId = searchParams.get("installationId") ?? "";

  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loadingOrganizations, setLoadingOrganizations] = useState(true);
  const [selectedOrganizationId, setSelectedOrganizationId] = useState<string>("");
  const [selectedIntegrationId, setSelectedIntegrationId] = useState<string>("");
  const [submitting, setSubmitting] = useState(false);

  const { data: connectedIntegrations = [], isLoading: loadingIntegrations } = useConnectedIntegrations(
    selectedOrganizationId,
    { enabled: !!selectedOrganizationId },
  );

  const sentryIntegrations = useMemo(() => {
    return (connectedIntegrations ?? []).filter((i: OrganizationsIntegration) => i.spec?.integrationName === "sentry");
  }, [connectedIntegrations]);

  useEffect(() => {
    if (accountLoading || !account) {
      setLoadingOrganizations(false);
      return;
    }

    const fetchOrganizations = async () => {
      try {
        const res = await fetch("/organizations", { credentials: "include" });
        if (!res.ok) {
          showErrorToast("Failed to load organizations");
          return;
        }
        const data = (await res.json()) as Organization[];
        setOrganizations(data);
        if (data.length === 1) {
          setSelectedOrganizationId(data[0].id);
        }
      } catch (_err) {
        showErrorToast("Failed to load organizations");
      } finally {
        setLoadingOrganizations(false);
      }
    };

    fetchOrganizations();
  }, [account, accountLoading]);

  const canSubmit = !!code && !!installationId && !!selectedOrganizationId && !!selectedIntegrationId && !submitting;

  const handleAttach = async () => {
    if (!canSubmit) return;

    setSubmitting(true);
    try {
      const res = await fetch("/api/v1/integrations/sentry/attach", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          "x-organization-id": selectedOrganizationId,
        },
        body: JSON.stringify({
          integrationId: selectedIntegrationId,
          installationId,
          code,
        }),
      });

      if (!res.ok) {
        showErrorToast("Failed to attach Sentry installation");
        return;
      }

      navigate(`/${selectedOrganizationId}/settings/integrations/${selectedIntegrationId}`, {
        state: { tab: "configuration" },
      });
    } catch (_err) {
      showErrorToast("Failed to attach Sentry installation");
    } finally {
      setSubmitting(false);
    }
  };

  if (!code || !installationId) {
    return (
      <div className="min-h-screen bg-slate-100 p-8">
        <div className="max-w-xl mx-auto bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <h1 className="text-xl font-semibold">Sentry setup</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
            Missing required parameters from Sentry redirect.
          </p>
        </div>
      </div>
    );
  }

  if (accountLoading) {
    return (
      <div className="min-h-screen bg-slate-100 p-8">
        <div className="max-w-xl mx-auto bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <h1 className="text-xl font-semibold">Finish Sentry installation</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">Loading...</p>
        </div>
      </div>
    );
  }

  if (!account) {
    const redirectParam = encodeURIComponent(
      `/sentry/setup?code=${encodeURIComponent(code)}&installationId=${encodeURIComponent(installationId)}`,
    );
    return (
      <div className="min-h-screen bg-slate-100 p-8">
        <div className="max-w-xl mx-auto bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <h1 className="text-xl font-semibold">Finish Sentry installation</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
            You need to log in to attach this Sentry installation to your SuperPlane organization.
          </p>
          <div className="pt-4 flex items-center justify-end gap-2">
            <Button variant="outline" onClick={() => navigate("/")}>
              Cancel
            </Button>
            <Button onClick={() => navigate(`/login?redirect=${redirectParam}`)}>Log in</Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-100 p-8">
      <div className="max-w-xl mx-auto bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
        <h1 className="text-xl font-semibold">Finish Sentry installation</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
          Select the SuperPlane organization and the Sentry integration installation you want to attach this Sentry App
          installation to.
        </p>

        <div className="mt-6 space-y-4">
          <div className="space-y-2">
            <Label>Organization</Label>
            <Select
              value={selectedOrganizationId}
              onValueChange={(v: string) => {
                setSelectedOrganizationId(v);
                setSelectedIntegrationId("");
              }}
              disabled={loadingOrganizations || organizations.length === 0}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder={loadingOrganizations ? "Loading..." : "Select an organization"} />
              </SelectTrigger>
              <SelectContent>
                {organizations.map((org) => (
                  <SelectItem key={org.id} value={org.id}>
                    {org.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Sentry integration</Label>
            <Select
              value={selectedIntegrationId}
              onValueChange={(v: string) => setSelectedIntegrationId(v)}
              disabled={!selectedOrganizationId || loadingIntegrations || sentryIntegrations.length === 0}
            >
              <SelectTrigger className="w-full">
                <SelectValue
                  placeholder={
                    !selectedOrganizationId
                      ? "Select an organization first"
                      : loadingIntegrations
                        ? "Loading..."
                        : sentryIntegrations.length === 0
                          ? "No Sentry integrations found in this org"
                          : "Select a Sentry integration"
                  }
                />
              </SelectTrigger>
              <SelectContent>
                {sentryIntegrations.map((integration) => (
                  <SelectItem key={integration.metadata?.id} value={integration.metadata?.id ?? ""}>
                    {integration.metadata?.name || integration.metadata?.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="pt-2 flex items-center justify-end gap-2">
            <Button variant="outline" onClick={() => navigate("/")}>
              Cancel
            </Button>
            <Button onClick={handleAttach} disabled={!canSubmit}>
              {submitting ? "Attaching..." : "Attach installation"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
