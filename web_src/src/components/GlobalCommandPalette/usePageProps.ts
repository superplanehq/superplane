import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { useOrganizationInviteLink } from "@/hooks/useOrganizationData";
import { useServiceAccounts } from "@/hooks/useServiceAccounts";
import { appPath } from "@/lib/appPaths";
import { Key, Palette, Plug } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { DOCS_URL } from "./constants";
import type { CommandPaletteModel } from "./model";
import type { CommandPalettePageProps } from "./pages";
import type { PaletteAction } from "./types";

export function useCommandPalettePageProps(model: CommandPaletteModel): CommandPalettePageProps {
  const [expandedSection, setExpandedSection] = useState<"apps" | "integrations" | null>(null);
  const organizationId = model.canvasListProps.organizationId ?? "";

  const { data: connectedIntegrations = [] } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: inviteLink } = useOrganizationInviteLink(organizationId, !!organizationId);
  const { data: serviceAccounts = [] } = useServiceAccounts(organizationId);

  const integrations = useMemo(
    () =>
      connectedIntegrations.map((i) => ({
        id: i.metadata?.id ?? "",
        name: i.metadata?.name ?? i.metadata?.integrationName ?? "Unknown",
        providerName: i.metadata?.integrationName ?? "",
        status: (i.status?.state === "ready" ? "ready" : i.status?.state === "error" ? "error" : "pending") as
          | "ready"
          | "error"
          | "pending",
      })),
    [connectedIntegrations],
  );

  const closePalette = useCallback(() => {
    model.setOpen(false);
    model.setSearch("");
    setExpandedSection(null);
  }, [model]);

  const searchResults = useMemo(
    () => buildSearchResults(model, integrations, serviceAccounts, organizationId, closePalette),
    [model, integrations, serviceAccounts, organizationId, closePalette],
  );

  const handleCopyInviteLink = useCallback(() => {
    if (!inviteLink?.token) {
      toast.error("Invite link not available");
      return;
    }
    const url = `${window.location.origin}/invite/${inviteLink.token}`;
    void navigator.clipboard.writeText(url).then(
      () => toast.success("Invite link copied"),
      () => toast.error("Failed to copy invite link"),
    );
    closePalette();
  }, [inviteLink, closePalette]);

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
    onExpandApps: () => setExpandedSection("apps"),
    onExpandIntegrations: () => setExpandedSection("integrations"),
    onCollapse: () => setExpandedSection(null),
    onGoToDocs: () => {
      closePalette();
      window.open(DOCS_URL, "_blank", "noopener,noreferrer");
    },
    onNewServiceAccount: () => {
      closePalette();
      window.location.href = `/${organizationId}/settings/service-accounts`;
    },
    onNewSecret: () => {
      closePalette();
      window.location.href = `/${organizationId}/settings/secrets`;
    },
    onSignOut: () => {
      closePalette();
      window.location.href = "/logout";
    },
    onConnectIntegration: () => {
      closePalette();
      window.location.href = `/${organizationId}/settings/integrations`;
    },
    onSelectIntegration: (id: string) => {
      closePalette();
      window.location.href = `/${organizationId}/settings/integrations/${id}`;
    },
    expandedSection,
    createAppLabel: "New App",
    createAppDisabled: !model.canvasListProps.organizationId,
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

function buildSearchResults(
  model: CommandPaletteModel,
  integrations: Array<{ id: string; name: string; providerName: string; status: string }>,
  serviceAccounts: Array<{ id?: string; name?: string }>,
  organizationId: string,
  closePalette: () => void,
): PaletteAction[] {
  if (!model.search) return [];
  const query = model.search.toLowerCase();
  const results: PaletteAction[] = [];

  for (const canvas of model.canvasListProps.canvases) {
    const name = canvas.metadata?.name ?? "";
    if (name.toLowerCase().includes(query)) {
      results.push({
        id: `app-${canvas.metadata?.id}`,
        label: name,
        description: "Open app",
        icon: Palette,
        onSelect: () => {
          const id = canvas.metadata?.id;
          if (organizationId && id) {
            closePalette();
            window.history.pushState({}, "", appPath(organizationId, id));
            window.dispatchEvent(new PopStateEvent("popstate"));
          }
        },
        keywords: [name],
      });
    }
  }

  for (const integration of integrations) {
    if (integration.name.toLowerCase().includes(query)) {
      results.push({
        id: `integration-${integration.id}`,
        label: integration.name,
        description: `Integration · ${integration.status}`,
        icon: Plug,
        onSelect: () => {
          closePalette();
          window.location.href = `/${organizationId}/settings/integrations/${integration.id}`;
        },
        keywords: [integration.name, integration.providerName],
      });
    }
  }

  for (const sa of serviceAccounts) {
    const name = sa.name ?? "";
    if (name.toLowerCase().includes(query)) {
      results.push({
        id: `sa-${sa.id}`,
        label: name,
        description: "Service Account",
        icon: Key,
        onSelect: () => {
          closePalette();
          window.location.href = `/${organizationId}/settings/service-accounts/${sa.id}`;
        },
        keywords: [name],
      });
    }
  }

  return results;
}
