import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useAvailableIntegrations, useCreateIntegration, useIntegrationResources } from "@/hooks/useIntegrations";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { useMemo, useRef, useState } from "react";

import { HomeIntegrationConnectRow, type IntegrationSelections } from "./InstallIntegrationsSection";
import { homeInstallPanelClassName } from "./homePageStyles";

const FACTORY_INTEGRATIONS = ["github", "claude"] as const;

interface FactorySetupPanelProps {
  organizationId?: string;
  busy?: boolean;
  onCancel: () => void;
  onInstall: (selections: IntegrationSelections, repository: string) => void;
  onPreviewWithoutConnecting: () => void;
}

export function FactorySetupPanel({
  organizationId: propOrgId,
  busy = false,
  onCancel,
  onInstall,
  onPreviewWithoutConnecting,
}: FactorySetupPanelProps) {
  const routeOrgId = useOrganizationId();
  const organizationId = propOrgId || routeOrgId || "";
  const [selections, setSelections] = useState<IntegrationSelections>({});
  const [connectedTools, setConnectedTools] = useState<Set<string>>(new Set());
  const [repository, setRepository] = useState("");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);
  const pendingConnectKeyRef = useRef<string | null>(null);

  const { data: availableIntegrations = [] } = useAvailableIntegrations({ enabled: !!organizationId });
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");

  const githubConnected = connectedTools.has("github");
  const allConnected = FACTORY_INTEGRATIONS.every((name) => connectedTools.has(name));
  const canInstall = !busy && allConnected && repository.trim() !== "";

  const dialogDefinition = useMemo(
    () => (dialogIntegrationName ? availableIntegrations.find((d) => d.name === dialogIntegrationName) : undefined),
    [availableIntegrations, dialogIntegrationName],
  );

  const defaultDialogName = useMemo(
    () => (dialogIntegrationName ? getNextIntegrationName(dialogIntegrationName, new Set()) : ""),
    [dialogIntegrationName],
  );

  const openConnectDialog = (integrationName: string) => {
    pendingConnectKeyRef.current = integrationName;
    setDialogIntegrationName(integrationName);
  };

  return (
    <div className={homeInstallPanelClassName} role="region" aria-label="Software Factory setup">
      <div className="mb-5">
        <h3 className="text-base font-medium text-slate-900 dark:text-gray-100">Connect your GitHub and Claude</h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-gray-400">
          This will create software factory that automates your delivery from trigger to pull request.
        </p>
      </div>

      <div className="mb-5">
        <div className="divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-gray-700/70 dark:border-gray-700/70">
          {FACTORY_INTEGRATIONS.map((name) => (
            <HomeIntegrationConnectRow
              key={name}
              name={name}
              connected={connectedTools.has(name)}
              onConnect={() => openConnectDialog(name)}
            />
          ))}
        </div>
      </div>

      {githubConnected && (
        <div className="mb-5 pt-4">
          <Label
            htmlFor="factory-repository"
            className="mb-2 block text-xs font-semibold text-slate-700 dark:text-gray-300"
          >
            Choose repository
          </Label>
          <FactoryRepositorySelect
            organizationId={organizationId}
            integrationId={selections.github?.id ?? ""}
            value={repository}
            onChange={setRepository}
          />
        </div>
      )}

      <div className="flex flex-wrap items-center pt-4">
        <div className="flex items-center gap-2.5">
          <Button type="button" disabled={!canInstall} onClick={() => onInstall(selections, repository.trim())}>
            Install
          </Button>
          <Button type="button" variant="outline" onClick={onCancel} disabled={busy}>
            Cancel
          </Button>
        </div>
        <button
          type="button"
          disabled={busy}
          onClick={onPreviewWithoutConnecting}
          className="ml-4 text-xs font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 disabled:opacity-50 dark:text-gray-200 dark:decoration-gray-600"
        >
          Let me preview the app without connecting
        </button>
      </div>

      <IntegrationCreateDialog
        open={!!dialogIntegrationName}
        onOpenChange={(open) => {
          if (!open) {
            setDialogIntegrationName(null);
            createIntegrationMutation.reset();
          }
        }}
        integrationDefinition={dialogDefinition ?? null}
        organizationId={organizationId}
        onCreateIntegration={async (payload) => {
          const res = await createIntegrationMutation.mutateAsync(payload);
          return res.data;
        }}
        onReset={() => createIntegrationMutation.reset()}
        defaultName={defaultDialogName}
        onCreated={(integrationId, instanceName) => {
          // Dialog calls onOpenChange(false) before onCreated; keep the key in a ref across that close.
          const key = pendingConnectKeyRef.current;
          pendingConnectKeyRef.current = null;
          if (key) {
            setConnectedTools((prev) => new Set(prev).add(key));
            setSelections((prev) => ({
              ...prev,
              [key]: { id: integrationId, name: instanceName },
            }));
            if (key === "github") {
              setRepository("");
            }
          }
          setDialogIntegrationName(null);
        }}
      />
    </div>
  );
}

function FactoryRepositorySelect({
  organizationId,
  integrationId,
  value,
  onChange,
}: {
  organizationId: string;
  integrationId: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const { data: resources = [], isLoading } = useIntegrationResources(organizationId, integrationId, "repository");
  const options = useMemo(
    () =>
      resources
        .map((resource) => {
          const name = resource.name?.trim();
          if (!name) return null;
          return { value: name, label: name };
        })
        .filter((option): option is { value: string; label: string } => option !== null),
    [resources],
  );

  return (
    <Select value={value || undefined} onValueChange={onChange} disabled={isLoading || options.length === 0}>
      <SelectTrigger id="factory-repository" className="w-full">
        <SelectValue placeholder={isLoading ? "Loading repositories…" : "Select a repository"} />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
