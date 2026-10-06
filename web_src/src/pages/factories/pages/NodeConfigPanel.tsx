import type {
  ConfigurationField,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
  SuperplaneComponentsNode,
} from "@/api-client";
import { useComponents } from "@/hooks/useComponentData";
import { useTriggers, useWidgets } from "@/hooks/useCanvasData";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { isAgentHarnessComponent } from "@/lib/agentRunnerSteps";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import { SettingsTab } from "@/ui/componentSidebar/SettingsTab";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { ChevronsRight } from "lucide-react";
import { useMemo, useState } from "react";
import { useLocation } from "react-router";

import { AgentRunnerSettings } from "./AgentRunnerSettings";
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
  factoryId,
  factoryKey,
  onClose,
  onSave,
}: {
  node: SuperplaneComponentsNode;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  onClose: () => void;
  onSave: (update: NodeConfigurationUpdate) => Promise<void> | void;
}) {
  const catalog = useNodeCatalog(node, organizationId);
  const settings = nodeSettingsProps(node, catalog.definition);
  const [connectOpen, setConnectOpen] = useState(false);

  return (
    <>
      <aside
        className="relative flex w-1/2 min-w-0 shrink-0 flex-col border-l border-border bg-background"
        aria-label={catalog.nodeName}
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
          <h2 className="truncate text-[15px] font-semibold text-foreground">{catalog.nodeName}</h2>
        </header>
        <div className="flex min-h-0 flex-1 flex-col">
          {catalog.loading ? (
            <p className="workspace-body-text px-10 py-6 text-muted-foreground">{NODE_CONFIG_COPY.loading}</p>
          ) : isAgentHarnessComponent(node.component) ? (
            <AgentRunnerSettings
              node={node}
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              onSave={onSave}
            />
          ) : (
            <SettingsTab
              layout="fill"
              chrome={settings.chrome}
              booleanControl={settings.booleanControl}
              fieldGroups={settings.fieldGroups}
              hiddenFieldNames={settings.hiddenFieldNames}
              leadingContent={settings.leadingContent}
              mode="edit"
              nodeId={node.id}
              nodeName={catalog.nodeName}
              nodeLabel={catalog.definition?.label}
              blockName={catalog.componentName}
              configuration={settings.configuration}
              configurationFields={settings.configurationFields}
              integrationName={catalog.integrationName}
              integrationRef={node.integration}
              integrationDefinition={integrationDefinitionFields(catalog.integrationDefinition)}
              integrations={catalog.integrations}
              domainId={organizationId}
              onOpenCreateIntegrationDialog={() => setConnectOpen(true)}
              showConcurrency={catalog.showConcurrency}
              concurrency={node.concurrency}
              concurrencyMaxOnly={catalog.concurrencyMaxOnly}
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
      {organizationId && catalog.integrationDefinition ? (
        <StepIntegrationConnect
          open={connectOpen}
          onOpenChange={setConnectOpen}
          organizationId={organizationId}
          integration={catalog.integrationDefinition}
          connectedIntegrations={catalog.integrations}
        />
      ) : null}
    </>
  );
}

function useNodeCatalog(node: SuperplaneComponentsNode, organizationId?: string) {
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
  const definition = findCatalogEntry(node, {
    components,
    triggers,
    widgets: widgetsQuery.data,
  });
  const integrationName = integrationNameForComponent(availableIntegrations.data, componentName);

  return {
    componentName,
    definition,
    integrationName,
    integrationDefinition: availableIntegrations.data?.find((integration) => integration.name === integrationName),
    integrations: connectedIntegrations.data ?? [],
    nodeName: nodeDisplayName(node),
    loading: catalogIsLoading(node, {
      components: componentsQuery.isLoading,
      triggers: triggersQuery.isLoading,
      widgets: widgetsQuery.isLoading,
      integrations: availableIntegrations.isLoading,
    }),
    showConcurrency: node.type === "TYPE_ACTION" && componentName !== "merge",
    concurrencyMaxOnly: componentName === "loop",
  };
}

function catalogIsLoading(
  node: SuperplaneComponentsNode,
  loading: { components: boolean; triggers: boolean; widgets: boolean; integrations: boolean },
) {
  if (loading.integrations) {
    return true;
  }
  return isCatalogLoading(node, loading);
}

function nodeDisplayName(node: SuperplaneComponentsNode) {
  const name = node.name?.trim();
  if (!name) {
    return "Step";
  }
  return name;
}

function nodeSettingsProps(node: SuperplaneComponentsNode, definition: CatalogEntry | undefined) {
  const configuration = node.configuration ?? {};
  if (node.component !== PULL_REQUEST_TRIGGER) {
    return {
      chrome: "full" as const,
      booleanControl: "switch" as const,
      fieldGroups: undefined,
      hiddenFieldNames: undefined,
      leadingContent: undefined,
      configuration,
      configurationFields: definition?.configuration ?? [],
    };
  }
  return {
    chrome: "fields" as const,
    booleanControl: "checkbox" as const,
    fieldGroups: PULL_REQUEST_FIELD_GROUPS,
    hiddenFieldNames: PULL_REQUEST_HIDDEN_FIELDS,
    leadingContent: repositoryLine(repositoryDisplay(configuration.repository)),
    configuration,
    configurationFields: presentPullRequestFields(definition?.configuration ?? [], configuration),
  };
}

function repositoryLine(label: string | undefined) {
  if (!label) {
    return undefined;
  }
  return (
    <div className="flex flex-col gap-1.5">
      <p className="text-sm font-medium text-foreground">{NODE_CONFIG_COPY.repository}</p>
      <p className="text-sm text-muted-foreground">{label}</p>
    </div>
  );
}

function StepIntegrationConnect({
  open,
  onOpenChange,
  organizationId,
  integration,
  connectedIntegrations,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  integration: IntegrationsIntegrationDefinition;
  connectedIntegrations: OrganizationsIntegration[];
}) {
  const createIntegration = useCreateIntegration(organizationId, "node_configuration");
  const location = useLocation();
  const existingNames = new Set(
    connectedIntegrations.map((item) => item.metadata?.name).filter((name): name is string => Boolean(name)),
  );

  return (
    <IntegrationCreateDialog
      open={open}
      onOpenChange={onOpenChange}
      integrationDefinition={integration}
      organizationId={organizationId}
      onCreateIntegration={async (payload) => {
        const response = await createIntegration.mutateAsync(payload);
        return response.data;
      }}
      onReset={() => createIntegration.reset()}
      defaultName={integration.label || integration.name || ""}
      existingIntegrationNames={existingNames}
      setupReturnTo={`${location.pathname}${location.search}`}
      integrationHomeHref={`/${organizationId}/settings/integrations`}
      onCreated={() => onOpenChange(false)}
    />
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
