import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useCallback, useMemo, useState } from "react";

import { IntegrationsSection, type IntegrationSelections } from "./InstallIntegrationsSection";
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
  const [repository, setRepository] = useState("");

  const githubConnected = Boolean(selections.github);
  const allConnected = FACTORY_INTEGRATIONS.every((name) => selections[name]);
  const canInstall = !busy && allConnected && repository.trim() !== "";

  const handleSelectionsChange = useCallback((next: IntegrationSelections) => {
    setSelections((prev) => {
      if (prev.github?.id !== next.github?.id) {
        setRepository("");
      }
      return next;
    });
  }, []);

  return (
    <div className={homeInstallPanelClassName} role="region" aria-label="Software Factory setup">
      <div className="mb-5">
        <h3 className="text-base font-medium text-slate-900 dark:text-gray-100">Connect your GitHub and Claude</h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-gray-400">
          This will create software factory that automates your delivery from trigger to pull request.
        </p>
      </div>

      <div className="mb-5">
        <IntegrationsSection
          integrations={[...FACTORY_INTEGRATIONS]}
          organizationId={organizationId}
          selections={selections}
          onSelectionsChange={handleSelectionsChange}
          variant="status"
        />
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
        <Button
          type="button"
          variant="link"
          disabled={busy}
          onClick={onPreviewWithoutConnecting}
          className="ml-4 h-auto p-0 text-xs font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 dark:text-gray-200 dark:decoration-gray-600"
        >
          Take me to the app without connecting
        </Button>
      </div>
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
  const {
    data: resources = [],
    isLoading,
    isError,
    refetch,
  } = useIntegrationResources(organizationId, integrationId, "repository");
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

  if (isError) {
    return (
      <div className="space-y-2">
        <p className="text-xs text-red-600 dark:text-red-400">Couldn't load repositories. Try again.</p>
        <Button type="button" variant="outline" size="sm" onClick={() => void refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  if (!isLoading && options.length === 0) {
    return (
      <p className="text-xs text-slate-500 dark:text-gray-400">
        No repositories found for this GitHub connection. Check access, then try again.
      </p>
    );
  }

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
