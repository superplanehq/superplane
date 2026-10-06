import type { ConfigurationField, IntegrationsIntegrationDefinition, SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useComponents } from "@/hooks/useComponentData";
import { useTriggers, useWidgets } from "@/hooks/useCanvasData";
import { useAvailableIntegrations, useConnectedIntegrations } from "@/hooks/useIntegrations";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import { SettingsTab } from "@/ui/componentSidebar/SettingsTab";
import { X } from "lucide-react";
import { useMemo } from "react";

import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

const NODE_CONFIG_COPY = {
  closeStep: "Close step",
  loading: "Loading settings.",
} as const;

type CatalogEntry = {
  name?: string;
  label?: string;
  configuration?: ConfigurationField[];
};

export function NodeConfigPanel({
  node,
  organizationId,
  onClose,
  onSave,
}: {
  node: SuperplaneComponentsNode;
  organizationId?: string;
  onClose: () => void;
  onSave: (update: NodeConfigurationUpdate) => Promise<void> | void;
}) {
  const componentName = node.component ?? "";
  const componentsQuery = useComponents(organizationId ?? "");
  const triggersQuery = useTriggers();
  const widgetsQuery = useWidgets();
  const availableIntegrations = useAvailableIntegrations({ organizationId });
  const connectedIntegrations = useConnectedIntegrations(organizationId ?? "", {
    enabled: Boolean(organizationId),
  });
  const components = useMemo(
    () => mergeCatalog(componentsQuery.data, availableIntegrations.data, actionsFromCapabilities),
    [availableIntegrations.data, componentsQuery.data],
  );
  const triggers = useMemo(
    () => mergeCatalog(triggersQuery.data, availableIntegrations.data, triggersFromCapabilities),
    [availableIntegrations.data, triggersQuery.data],
  );
  const catalogLoading =
    isCatalogLoading(node, {
      components: componentsQuery.isLoading,
      triggers: triggersQuery.isLoading,
      widgets: widgetsQuery.isLoading,
    }) || availableIntegrations.isLoading;
  const definition = findCatalogEntry(node, {
    components,
    triggers,
    widgets: widgetsQuery.data,
  });
  const integrationName = integrationNameForComponent(availableIntegrations.data, componentName);
  const integrationDefinition = availableIntegrations.data?.find((integration) => integration.name === integrationName);
  const nodeName = node.name?.trim() || "Step";

  return (
    <aside
      className="flex w-1/2 min-w-0 shrink-0 flex-col border-l border-border"
      aria-label={nodeName}
      data-testid="merge-confidence-config-form"
    >
      <header className="flex shrink-0 items-center justify-between gap-3 border-b border-border px-5 py-3">
        <h2 className="truncate text-[15px] font-semibold text-foreground">{nodeName}</h2>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          onClick={onClose}
          aria-label={NODE_CONFIG_COPY.closeStep}
          data-testid="merge-confidence-config-form-close"
        >
          <X aria-hidden />
        </Button>
      </header>
      <div className="flex min-h-0 flex-1 flex-col">
        {catalogLoading ? (
          <p className="workspace-body-text px-6 py-6 text-muted-foreground">{NODE_CONFIG_COPY.loading}</p>
        ) : (
          <SettingsTab
            layout="fill"
            mode="edit"
            nodeId={node.id}
            nodeName={nodeName}
            nodeLabel={definition?.label}
            blockName={componentName}
            configuration={node.configuration ?? {}}
            configurationFields={definition?.configuration ?? []}
            integrationName={integrationName}
            integrationRef={node.integration}
            integrationDefinition={integrationDefinitionFields(integrationDefinition)}
            integrations={connectedIntegrations.data ?? []}
            domainId={organizationId}
            showConcurrency={node.type === "TYPE_ACTION" && componentName !== "merge"}
            concurrency={node.concurrency}
            concurrencyMaxOnly={componentName === "loop"}
            onSave={(configuration, name, integration, concurrency) =>
              onSave({
                nodeId: node.id ?? "",
                name,
                configuration,
                integration,
                concurrency,
              })
            }
          />
        )}
      </div>
    </aside>
  );
}

function mergeCatalog(
  entries: CatalogEntry[] | undefined,
  integrations: IntegrationsIntegrationDefinition[] | undefined,
  fromCapabilities: (capabilities: NonNullable<IntegrationsIntegrationDefinition["capabilities"]>) => CatalogEntry[],
): CatalogEntry[] {
  const merged = [...(entries ?? [])];
  for (const integration of integrations ?? []) {
    if (integration.capabilities) {
      merged.push(...fromCapabilities(integration.capabilities));
    }
  }
  return merged;
}

function isCatalogLoading(
  node: SuperplaneComponentsNode,
  loading: { components: boolean; triggers: boolean; widgets: boolean },
): boolean {
  if (node.type === "TYPE_TRIGGER") {
    return loading.triggers;
  }
  if (node.type === "TYPE_WIDGET") {
    return loading.widgets;
  }
  return loading.components;
}

function findCatalogEntry(
  node: SuperplaneComponentsNode,
  catalogs: { components?: CatalogEntry[]; triggers?: CatalogEntry[]; widgets?: CatalogEntry[] },
): CatalogEntry | undefined {
  const entries =
    node.type === "TYPE_TRIGGER"
      ? catalogs.triggers
      : node.type === "TYPE_WIDGET"
        ? catalogs.widgets
        : catalogs.components;
  return entries?.find((entry) => entry.name === node.component);
}

function integrationNameForComponent(
  integrations: IntegrationsIntegrationDefinition[] | undefined,
  componentName: string,
): string | undefined {
  if (!componentName) {
    return undefined;
  }
  return integrations?.find((integration) =>
    integration.capabilities?.some((capability) => capability.name === componentName),
  )?.name;
}

function integrationDefinitionFields(
  integration: IntegrationsIntegrationDefinition | undefined,
): { name?: string; label?: string; icon?: string } | undefined {
  if (!integration) {
    return undefined;
  }
  return { name: integration.name, label: integration.label, icon: integration.icon };
}
