import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { useAPIKeys } from "@/hooks/useApiKeys";
import { appPath } from "@/lib/appPaths";
import { Key, Palette, Plug } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { DOCS_URL } from "./constants";
import type { CommandPaletteModel } from "./model";
import type { CommandPalettePageProps, IntegrationItem, IntegrationStatus } from "./pages";
import type { PaletteAction } from "./types";

export function useCommandPalettePageProps(model: CommandPaletteModel): CommandPalettePageProps {
  const [expandedSection, setExpandedSection] = useState<"apps" | "integrations" | null>(null);
  const organizationId = model.canvasListProps.organizationId ?? "";
  const goTo = model.canvasListProps.goTo;
  const canFetchIntegrations = canFetchPaletteData(organizationId, model.canReadIntegrations);
  const canFetchAPIKeys = canFetchPaletteData(organizationId, model.canReadAPIKeys);

  const { data: connectedIntegrations = [], isLoading: integrationsLoading } = useConnectedIntegrations(
    organizationId,
    {
      enabled: canFetchIntegrations,
    },
  );
  const { data: apiKeys = [], isLoading: apiKeysLoading } = useAPIKeys(organizationId, {
    enabled: canFetchAPIKeys,
  });

  const integrations = useMemo<IntegrationItem[]>(
    () =>
      connectedIntegrations.map((integration) => ({
        id: integration.metadata?.id ?? "",
        name: integration.metadata?.name ?? integration.metadata?.integrationName ?? "Unknown",
        providerName: integration.metadata?.integrationName ?? "",
        status: integrationStatusFrom(integration.status?.state),
      })),
    [connectedIntegrations],
  );

  const closePalette = useCallback(() => {
    model.setOpen(false);
    model.setSearch("");
    setExpandedSection(null);
  }, [model]);

  const searchResults = useMemo(
    () => buildSearchResults(model, integrations, apiKeys, organizationId, goTo),
    [model, integrations, apiKeys, organizationId, goTo],
  );

  const handleSetSearch = useCallback(
    (value: string) => {
      model.setSearch(value);
      if (value) setExpandedSection(null);
    },
    [model],
  );

  return {
    canvasListProps: model.canvasListProps,
    integrations,
    onCreateApp: () => {
      const createAction = model.rootActions.find((a) => a.id === "new-canvas");
      createAction?.onSelect?.();
    },
    onInviteMember: () => goTo(`/${organizationId}/settings/members`),
    showInviteMember: model.canInviteMembers,
    onExpandApps: () => setExpandedSection("apps"),
    onExpandIntegrations: () => setExpandedSection("integrations"),
    onCollapse: () => setExpandedSection(null),
    onGoToDocs: () => {
      closePalette();
      window.open(DOCS_URL, "_blank", "noopener,noreferrer");
    },
    onNewAPIKey: () => {
      goTo(`/${organizationId}/settings/api-keys`);
    },
    onNewSecret: () => {
      goTo(`/${organizationId}/settings/secrets`);
    },
    onSignOut: () => {
      closePalette();
      window.location.href = "/logout";
    },
    onConnectIntegration: () => {
      goTo(`/${organizationId}/settings/integrations`);
    },
    onSelectIntegration: (id: string) => {
      goTo(`/${organizationId}/settings/integrations/${id}`);
    },
    expandedSection,
    createAppLabel: model.rootActions.find((a) => a.id === "new-canvas")?.label ?? "New App",
    createAppDisabled: model.rootActions.find((a) => a.id === "new-canvas")?.disabled ?? true,
    searchActive: !!model.search,
    searchLoading: [model.canvasListProps.canvasesLoading, integrationsLoading, apiKeysLoading].some(Boolean),
    searchResults,
    handleSetSearch,
    handleOpenChange: (open: boolean) => {
      model.setOpen(open);
      if (!open) {
        model.setSearch("");
        setExpandedSection(null);
      }
    },
  };
}

function canFetchPaletteData(organizationId: string, allowed: boolean) {
  return Boolean(organizationId) && allowed;
}

type APIKeySearchItem = {
  id?: string;
  name?: string;
};

function integrationStatusFrom(state: string | undefined): IntegrationStatus {
  if (state === "ready") return "ready";
  if (state === "error") return "error";
  return "pending";
}

function buildSearchResults(
  model: CommandPaletteModel,
  integrations: IntegrationItem[],
  apiKeys: APIKeySearchItem[],
  organizationId: string,
  goTo: (href: string) => void,
): PaletteAction[] {
  if (!model.search) return [];
  const query = model.search.toLowerCase();
  const results: PaletteAction[] = [...model.canvasNodeSearchActions];

  for (const canvas of model.canvasListProps.canvases) {
    const name = canvas.name ?? "";
    const description = canvas.description ?? "";
    if (matchesSearch(query, name, description)) {
      results.push({
        id: `app-${canvas.id}`,
        label: name,
        description: "Open app",
        icon: Palette,
        onSelect: () => {
          const id = canvas.id;
          if (organizationId && id) {
            goTo(appPath(organizationId, id));
          }
        },
        keywords: [name, description],
      });
    }
  }

  for (const integration of integrations) {
    if (matchesSearch(query, integration.name, integration.providerName, integration.status)) {
      results.push({
        id: `integration-${integration.id}`,
        label: integration.name,
        description: `Integration · ${integration.status}`,
        icon: Plug,
        onSelect: () => goTo(`/${organizationId}/settings/integrations/${integration.id}`),
        keywords: [integration.name, integration.providerName, integration.status],
      });
    }
  }

  for (const apiKey of apiKeys) {
    const name = apiKey.name ?? "";
    if (matchesSearch(query, name, apiKey.id)) {
      results.push({
        id: `api-key-${apiKey.id}`,
        label: name,
        description: "API Key",
        icon: Key,
        onSelect: () => goTo(`/${organizationId}/settings/api-keys/${apiKey.id}`),
        keywords: [name, apiKey.id ?? ""],
      });
    }
  }

  return results;
}

function matchesSearch(query: string, ...values: Array<string | undefined>) {
  return values.some((value) => value?.toLowerCase().includes(query));
}
