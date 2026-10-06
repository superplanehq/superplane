import type { ConfigurationField, IntegrationsIntegrationDefinition, SuperplaneComponentsNode } from "@/api-client";
import { useComponents } from "@/hooks/useComponentData";
import { useTriggers, useWidgets } from "@/hooks/useCanvasData";
import { useAvailableIntegrations, useConnectedIntegrations } from "@/hooks/useIntegrations";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import { SettingsTab } from "@/ui/componentSidebar/SettingsTab";
import { ChevronsRight } from "lucide-react";
import { useMemo } from "react";

import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

const PULL_REQUEST_TRIGGER = "github.onPullRequest";
const PULL_REQUEST_HIDDEN_FIELDS = ["repository", "customName"] as const;

const ONLY_FACTORY_PULL_REQUESTS = "onlyFactoryPullRequests";
const PULL_REQUEST_ACTIONS_FIELD = "actions";

const PULL_REQUEST_ACTIONS = [
  { value: "opened", label: "A pull request is opened" },
  { value: "synchronize", label: "New commits are pushed" },
  { value: "reopened", label: "A pull request is reopened" },
  { value: "ready_for_review", label: "A pull request is ready for review" },
] as const;

const NODE_CONFIG_COPY = {
  collapseStep: "Collapse step",
  loading: "Loading settings.",
  repository: "Repository",
  applicationRepository: "Application repository",
  onlyFactoryPullRequests: "Run only when this factory created the pull request",
  startRunWhen: "Start a run when:",
  filters: "Filters",
} as const;

const PULL_REQUEST_FIELD_GROUPS = [
  { label: NODE_CONFIG_COPY.startRunWhen, fieldNames: [PULL_REQUEST_ACTIONS_FIELD] },
  { label: NODE_CONFIG_COPY.filters, fieldNames: ["ignoreDrafts", ONLY_FACTORY_PULL_REQUESTS] },
] as const;

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
  const simplifiedPullRequest = node.component === PULL_REQUEST_TRIGGER;
  const repositoryLabel = simplifiedPullRequest ? repositoryDisplay(node.configuration?.repository) : undefined;

  return (
    <aside
      className="relative flex w-1/2 min-w-0 shrink-0 flex-col border-l border-border bg-background"
      aria-label={nodeName}
      data-testid="merge-confidence-config-form"
    >
      <button
        type="button"
        onClick={onClose}
        aria-label={NODE_CONFIG_COPY.collapseStep}
        data-testid="merge-confidence-config-form-collapse"
        className="absolute top-5 left-0 z-10 flex h-8 w-7 -translate-x-[calc(100%-1px)] items-center justify-center rounded-l-md border border-r-0 border-border bg-background text-muted-foreground hover:bg-accent"
      >
        <ChevronsRight className="size-4" aria-hidden />
      </button>
      <header className="flex shrink-0 items-center px-10 pt-6">
        <h2 className="truncate text-[15px] font-semibold text-foreground">{nodeName}</h2>
      </header>
      <div className="flex min-h-0 flex-1 flex-col">
        {catalogLoading ? (
          <p className="workspace-body-text px-10 py-6 text-muted-foreground">{NODE_CONFIG_COPY.loading}</p>
        ) : (
          <SettingsTab
            layout="fill"
            chrome={simplifiedPullRequest ? "fields" : "full"}
            booleanControl={simplifiedPullRequest ? "checkbox" : "switch"}
            fieldGroups={simplifiedPullRequest ? PULL_REQUEST_FIELD_GROUPS : undefined}
            hiddenFieldNames={simplifiedPullRequest ? PULL_REQUEST_HIDDEN_FIELDS : undefined}
            leadingContent={
              repositoryLabel ? (
                <div className="flex flex-col gap-1.5">
                  <p className="text-sm font-medium text-foreground">{NODE_CONFIG_COPY.repository}</p>
                  <p className="text-sm text-muted-foreground">{repositoryLabel}</p>
                </div>
              ) : undefined
            }
            mode="edit"
            nodeId={node.id}
            nodeName={nodeName}
            nodeLabel={definition?.label}
            blockName={componentName}
            configuration={node.configuration ?? {}}
            configurationFields={
              simplifiedPullRequest
                ? presentPullRequestFields(definition?.configuration ?? [], node.configuration ?? {})
                : (definition?.configuration ?? [])
            }
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

function presentPullRequestFields(
  fields: ConfigurationField[],
  configuration: Record<string, unknown>,
): ConfigurationField[] {
  return fields.map((field) => {
    if (field.name === ONLY_FACTORY_PULL_REQUESTS) {
      return { ...field, label: NODE_CONFIG_COPY.onlyFactoryPullRequests, description: undefined };
    }
    if (field.name !== PULL_REQUEST_ACTIONS_FIELD) {
      return field;
    }
    return {
      ...field,
      description: undefined,
      typeOptions: {
        ...field.typeOptions,
        multiSelect: {
          ...field.typeOptions?.multiSelect,
          options: pullRequestActionOptions(field, configuration.actions),
        },
      },
    };
  });
}

function pullRequestActionOptions(field: ConfigurationField, selected: unknown): { label: string; value: string }[] {
  const catalog = new Map(
    (field.typeOptions?.multiSelect?.options ?? [])
      .filter((option) => option.value)
      .map((option) => [option.value as string, option.label || option.value || ""]),
  );
  const chosen = Array.isArray(selected) ? selected.filter((item): item is string => typeof item === "string") : [];
  const shown = new Set<string>(PULL_REQUEST_ACTIONS.map((action) => action.value));
  return [
    ...PULL_REQUEST_ACTIONS.map((action) => ({ value: action.value, label: action.label })),
    ...chosen.filter((value) => !shown.has(value)).map((value) => ({ value, label: catalog.get(value) || value })),
  ];
}

function repositoryDisplay(value: unknown): string | undefined {
  if (typeof value !== "string" || value.trim() === "") {
    return undefined;
  }
  if (value.includes("{{")) {
    return NODE_CONFIG_COPY.applicationRepository;
  }
  return value;
}

function integrationDefinitionFields(
  integration: IntegrationsIntegrationDefinition | undefined,
): { name?: string; label?: string; icon?: string } | undefined {
  if (!integration) {
    return undefined;
  }
  return { name: integration.name, label: integration.label, icon: integration.icon };
}
