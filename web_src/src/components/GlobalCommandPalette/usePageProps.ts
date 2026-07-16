import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { useOrganizationInviteLink } from "@/hooks/useOrganizationData";
import { useServiceAccounts } from "@/hooks/useServiceAccounts";
import { appPath } from "@/lib/appPaths";
import { Key, Palette, Plug } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { DOCS_URL } from "./constants";
import type { CommandPaletteModel } from "./model";
import type { CommandPalettePageProps, IntegrationItem, IntegrationStatus } from "./pages";
import type { PaletteAction } from "./types";

export function useCommandPalettePageProps(model: CommandPaletteModel): CommandPalettePageProps {
  const [expandedSection, setExpandedSection] = useState<"apps" | "integrations" | null>(null);
  const organizationId = model.canvasListProps.organizationId ?? "";
  const goTo = model.canvasListProps.goTo;

  // Reset expanded section when palette closes
  useEffect(() => {
    if (!model.open) setExpandedSection(null);
  }, [model.open]);

  const { data: connectedIntegrations = [] } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: inviteLink } = useOrganizationInviteLink(organizationId, !!organizationId && model.canManageInviteLink);
  const { data: serviceAccounts = [] } = useServiceAccounts(organizationId);
  const inviteLinkToken = inviteLink?.enabled ? inviteLink.token : undefined;

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
    () => buildSearchResults(model, integrations, serviceAccounts, organizationId, goTo),
    [model, integrations, serviceAccounts, organizationId, goTo],
  );

  const handleCopyInviteLink = useCallback(() => {
    if (!inviteLinkToken) {
      toast.error("Invite link not available");
      return;
    }
    const url = `${window.location.origin}/invite/${inviteLinkToken}`;
    void navigator.clipboard.writeText(url).then(
      () => {
        toast.success("Invite link copied");
        closePalette();
      },
      () => toast.error("Failed to copy invite link"),
    );
  }, [inviteLinkToken, closePalette]);

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
    onCopyInviteLink: handleCopyInviteLink,
    showCopyInviteLink: model.canManageInviteLink,
    copyInviteLinkDisabled: !inviteLinkToken,
    onExpandApps: () => setExpandedSection("apps"),
    onExpandIntegrations: () => setExpandedSection("integrations"),
    onCollapse: () => setExpandedSection(null),
    onGoToDocs: () => {
      closePalette();
      window.open(DOCS_URL, "_blank", "noopener,noreferrer");
    },
    onNewServiceAccount: () => {
      goTo(`/${organizationId}/settings/service-accounts`);
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

type ServiceAccountSearchItem = {
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
  serviceAccounts: ServiceAccountSearchItem[],
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

  for (const sa of serviceAccounts) {
    const name = sa.name ?? "";
    if (matchesSearch(query, name, sa.id)) {
      results.push({
        id: `sa-${sa.id}`,
        label: name,
        description: "API Key",
        icon: Key,
        onSelect: () => goTo(`/${organizationId}/settings/service-accounts/${sa.id}`),
        keywords: [name, sa.id ?? ""],
      });
    }
  }

  return results;
}

function matchesSearch(query: string, ...values: Array<string | undefined>) {
  return values.some((value) => value?.toLowerCase().includes(query));
}
