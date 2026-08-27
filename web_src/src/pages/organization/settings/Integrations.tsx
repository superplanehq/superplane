import { Loader2, Plug, Search, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
} from "../../../hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/usePermissions";
import { ConfigurationFieldRenderer } from "../../../ui/configurationFieldRenderer";
import type {
  IntegrationsIntegrationDefinition,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "../../../api-client/types.gen";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/lib/usageLimits";
import { Icon } from "@/components/Icon";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { showErrorToast } from "@/lib/toast";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { analytics } from "@/lib/analytics";
import { isCapabilityBasedIntegrationDefinition, usesHostedGitHubAppInstall } from "@/lib/integrations";
import { startDirectGitHubConnect } from "@/lib/startDirectGitHubConnect";
import { GitHubConnectControls } from "./GitHubConnectControls";
import { posthog, isPostHogEnabled } from "@/posthog";
import { cn } from "@/lib/utils";
import { getNextIntegrationName } from "./components/IntegrationSetup/lib";
import { IntegrationInstanceRow } from "./IntegrationInstanceRow";
import {
  settingsEmptyStateIconClassName,
  settingsEmptyStateTitleClassName,
  settingsModalClassName,
  settingsPanelClassName,
} from "./settingsPageStyles";
import { useIntegrationCatalog } from "./useIntegrationCatalog";

const INTEGRATION_SURVEY_NAME = "Integration Survey";

function resetConnectModal(args: {
  resetMutation: () => void;
  setIsModalOpen: (open: boolean) => void;
  setSelectedIntegration: (value: IntegrationsIntegrationDefinition | null) => void;
  setIntegrationName: (value: string) => void;
  setConfiguration: (value: Record<string, unknown>) => void;
}) {
  args.setIsModalOpen(false);
  args.setSelectedIntegration(null);
  args.setIntegrationName("");
  args.setConfiguration({});
  args.resetMutation();
}

function connectIntegrationDefinition(args: {
  integration: IntegrationsIntegrationDefinition;
  canCreateIntegrations: boolean;
  organizationId: string;
  integrationNames: Set<string>;
  organizationIntegrations: OrganizationsIntegration[];
  currentUserId?: string;
  navigate: ReturnType<typeof useNavigate>;
  mutateAsync: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
  setSelectedIntegration: (value: IntegrationsIntegrationDefinition | null) => void;
  setIntegrationName: (value: string) => void;
  setConfiguration: (value: Record<string, unknown>) => void;
  setIsModalOpen: (open: boolean) => void;
}) {
  if (!args.canCreateIntegrations) return;

  if (usesHostedGitHubAppInstall(args.integration)) {
    analytics.integrationConnectStart("github", "integrations_page", args.organizationId);
    void startDirectGitHubConnect({
      organizationId: args.organizationId,
      returnTo: `/${args.organizationId}/settings/integrations`,
      existingNames: args.integrationNames,
      connected: args.organizationIntegrations,
      currentUserId: args.currentUserId,
      goTo: args.navigate,
      create: async (payload) => {
        const response = await args.mutateAsync(payload);
        return response.data;
      },
    }).catch((error) => {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to connect GitHub"));
    });
    return;
  }

  if (isCapabilityBasedIntegrationDefinition(args.integration)) {
    if (!args.integration.name) return;
    analytics.integrationConnectStart(args.integration.name, "integrations_page", args.organizationId);
    args.navigate(`/${args.organizationId}/settings/integrations/${args.integration.name}/setup`);
    return;
  }

  args.setSelectedIntegration(args.integration);
  args.setIntegrationName(getNextIntegrationName(args.integration.name ?? "integration", args.integrationNames));
  args.setConfiguration({});
  args.setIsModalOpen(true);
  analytics.integrationConnectStart(args.integration.name ?? "", "integrations_page", args.organizationId);
}

async function submitLegacyIntegrationConnect(args: {
  canCreateIntegrations: boolean;
  selectedIntegration: IntegrationsIntegrationDefinition | null;
  integrationName: string;
  configuration: Record<string, unknown>;
  organizationId: string;
  navigate: ReturnType<typeof useNavigate>;
  mutateAsync: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
  setIsModalOpen: (open: boolean) => void;
  setSelectedIntegration: (value: IntegrationsIntegrationDefinition | null) => void;
  setIntegrationName: (value: string) => void;
  setConfiguration: (value: Record<string, unknown>) => void;
}) {
  if (!args.canCreateIntegrations) return;
  if (!args.selectedIntegration?.name) return;

  try {
    const result = await args.mutateAsync({
      integrationName: args.selectedIntegration.name,
      name: args.integrationName,
      configuration: args.configuration,
    });
    args.setIsModalOpen(false);
    args.setSelectedIntegration(null);
    args.setIntegrationName("");
    args.setConfiguration({});

    if (result.data?.integration?.metadata?.id) {
      args.navigate(`/${args.organizationId}/settings/integrations/${result.data.integration.metadata.id}`);
    }
  } catch (_error) {
    showErrorToast(getUsageLimitToastMessage(_error, "Failed to create integration"));
  }
}

function IntegrationConnectModal(props: {
  selectedIntegration: IntegrationsIntegrationDefinition;
  selectedInstructions: string;
  integrationName: string;
  configuration: Record<string, unknown>;
  organizationId: string;
  canCreateIntegrations: boolean;
  isPending: boolean;
  isError: boolean;
  error: unknown;
  createIntegrationNotice: ReturnType<typeof getUsageLimitNotice>;
  onNameChange: (value: string) => void;
  onConfigurationChange: (value: Record<string, unknown>) => void;
  onConnect: () => void;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className={cn(settingsModalClassName, "max-h-[80vh] max-w-2xl overflow-y-auto")}>
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <IntegrationIcon
                integrationName={props.selectedIntegration.name}
                iconSlug={props.selectedIntegration.icon}
                className="w-6 h-6 text-gray-500 dark:text-gray-400"
              />
              <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">
                Connect {props.selectedIntegration.label || props.selectedIntegration.name}
              </h3>
            </div>
            <button
              onClick={props.onClose}
              className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
              disabled={props.isPending}
            >
              <Icon name="x" size="sm" />
            </button>
          </div>

          <div className="space-y-4">
            {props.selectedInstructions ? <IntegrationInstructions description={props.selectedInstructions} /> : null}
            <div>
              <Label className="text-gray-800 dark:text-gray-100 mb-2">
                Integration Name
                <span className="ml-1 text-gray-800 dark:text-gray-100">*</span>
              </Label>
              <Input
                type="text"
                value={props.integrationName}
                onChange={(e) => props.onNameChange(e.target.value)}
                placeholder="e.g., my-app-integration"
                required
                disabled={!props.canCreateIntegrations}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">A unique name for this integration</p>
            </div>

            {props.selectedIntegration.configuration && props.selectedIntegration.configuration.length > 0 ? (
              <div className="space-y-4">
                {props.selectedIntegration.configuration
                  .filter((field) => Boolean(field.name))
                  .map((field) => (
                    <ConfigurationFieldRenderer
                      key={field.name!}
                      field={field}
                      value={props.configuration[field.name!]}
                      onChange={(value) =>
                        props.onConfigurationChange({ ...props.configuration, [field.name!]: value })
                      }
                      allValues={props.configuration}
                      organizationId={props.organizationId}
                    />
                  ))}
              </div>
            ) : null}
          </div>

          <div className="flex justify-start gap-3 mt-6">
            <Button
              color="blue"
              onClick={props.onConnect}
              disabled={props.isPending || !props.integrationName?.trim() || !props.canCreateIntegrations}
              className="flex items-center gap-2"
            >
              {props.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Connecting...
                </>
              ) : (
                "Connect"
              )}
            </Button>
            <Button variant="outline" onClick={props.onClose} disabled={props.isPending}>
              Cancel
            </Button>
          </div>

          {props.isError && props.createIntegrationNotice ? (
            <UsageLimitAlert notice={props.createIntegrationNotice} className="mt-4" />
          ) : null}
          {props.isError && !props.createIntegrationNotice ? (
            <Alert variant="destructive" className="mt-4">
              <AlertTitle>Unable to create integration</AlertTitle>
              <AlertDescription>Failed to create integration: {getApiErrorMessage(props.error)}</AlertDescription>
            </Alert>
          ) : null}
        </div>
      </div>
    </div>
  );
}

interface IntegrationsProps {
  organizationId: string;
}

export function Integrations({ organizationId }: IntegrationsProps) {
  usePageTitle(["Integrations"]);
  const navigate = useNavigate();
  const { data: me } = useMe();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationDefinition | null>(null);
  const [integrationName, setIntegrationName] = useState("");
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [filterQuery, setFilterQuery] = useState("");
  const [isIntegrationSurveyActive, setIsIntegrationSurveyActive] = useState(false);
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");

  useEffect(() => {
    if (!isPostHogEnabled) return;

    posthog.getSurveys((surveys) => {
      const isActive = surveys.some(
        (survey) => survey.name === INTEGRATION_SURVEY_NAME && survey.start_date != null && survey.end_date == null,
      );
      setIsIntegrationSurveyActive(isActive);
    }, true);
  }, []);

  const { data: availableIntegrations = [], isLoading: loadingAvailable } = useAvailableIntegrations();
  const { data: organizationIntegrations = [], isLoading: loadingInstalled } = useConnectedIntegrations(organizationId);
  const createIntegrationMutation = useCreateIntegration(organizationId, "integrations_page");

  const isLoading = loadingAvailable || loadingInstalled;

  useReportPageReady(!isLoading && !permissionsLoading);

  const { integrationNames, integrationCatalog, filteredIntegrationCatalog } = useIntegrationCatalog(
    availableIntegrations,
    organizationIntegrations,
    filterQuery,
  );

  const selectedInstructions = selectedIntegration?.instructions?.trim() ?? "";

  const handleConnectClick = (integration: IntegrationsIntegrationDefinition) => {
    connectIntegrationDefinition({
      integration,
      canCreateIntegrations,
      organizationId,
      integrationNames,
      organizationIntegrations,
      currentUserId: me?.id,
      navigate,
      mutateAsync: createIntegrationMutation.mutateAsync,
      setSelectedIntegration,
      setIntegrationName,
      setConfiguration,
      setIsModalOpen,
    });
  };
  const handleConnect = async () => {
    await submitLegacyIntegrationConnect({
      canCreateIntegrations,
      selectedIntegration,
      integrationName,
      configuration,
      organizationId,
      navigate,
      mutateAsync: createIntegrationMutation.mutateAsync,
      setIsModalOpen,
      setSelectedIntegration,
      setIntegrationName,
      setConfiguration,
    });
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex justify-center items-center h-32">
          <p className="text-gray-500 dark:text-gray-400">Loading integrations...</p>
        </div>
      </div>
    );
  }

  return (
    <IntegrationsLoaded
      organizationId={organizationId}
      filterQuery={filterQuery}
      setFilterQuery={setFilterQuery}
      integrationCatalog={integrationCatalog}
      filteredIntegrationCatalog={filteredIntegrationCatalog}
      isIntegrationSurveyActive={isIntegrationSurveyActive}
      canCreateIntegrations={canCreateIntegrations}
      canUpdateIntegrations={canUpdateIntegrations}
      permissionsLoading={permissionsLoading}
      onConnectClick={handleConnectClick}
      isModalOpen={isModalOpen}
      selectedIntegration={selectedIntegration}
      selectedInstructions={selectedInstructions}
      integrationName={integrationName}
      configuration={configuration}
      createIntegrationMutation={createIntegrationMutation}
      onNameChange={setIntegrationName}
      onConfigurationChange={setConfiguration}
      onConnect={() => {
        void handleConnect();
      }}
      onCloseModal={() =>
        resetConnectModal({
          resetMutation: createIntegrationMutation.reset,
          setIsModalOpen,
          setSelectedIntegration,
          setIntegrationName,
          setConfiguration,
        })
      }
    />
  );
}

function IntegrationsLoaded(props: {
  organizationId: string;
  filterQuery: string;
  setFilterQuery: (value: string) => void;
  integrationCatalog: ReturnType<typeof useIntegrationCatalog>["integrationCatalog"];
  filteredIntegrationCatalog: ReturnType<typeof useIntegrationCatalog>["filteredIntegrationCatalog"];
  isIntegrationSurveyActive: boolean;
  canCreateIntegrations: boolean;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  onConnectClick: (integration: IntegrationsIntegrationDefinition) => void;
  isModalOpen: boolean;
  selectedIntegration: IntegrationsIntegrationDefinition | null;
  selectedInstructions: string;
  integrationName: string;
  configuration: Record<string, unknown>;
  createIntegrationMutation: ReturnType<typeof useCreateIntegration>;
  onNameChange: (value: string) => void;
  onConfigurationChange: (value: Record<string, unknown>) => void;
  onConnect: () => void;
  onCloseModal: () => void;
}) {
  const {
    organizationId,
    filterQuery,
    setFilterQuery,
    integrationCatalog,
    filteredIntegrationCatalog,
    isIntegrationSurveyActive,
    canCreateIntegrations,
    canUpdateIntegrations,
    permissionsLoading,
    onConnectClick,
    isModalOpen,
    selectedIntegration,
    selectedInstructions,
    integrationName,
    configuration,
    createIntegrationMutation,
    onNameChange,
    onConfigurationChange,
    onConnect,
    onCloseModal,
  } = props;
  const createIntegrationNotice = createIntegrationMutation.isError
    ? getUsageLimitNotice(createIntegrationMutation.error, organizationId)
    : null;

  return (
    <div className="pt-6">
      <div className="relative mb-4">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 dark:text-gray-400" />
        <Input
          type="text"
          value={filterQuery}
          onChange={(e) => setFilterQuery(e.target.value)}
          placeholder="Filter integrations..."
          className="pl-9 pr-9"
        />
        {filterQuery.length > 0 ? (
          <button
            type="button"
            onClick={() => setFilterQuery("")}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            aria-label="Clear filter"
          >
            <X className="w-4 h-4" />
          </button>
        ) : null}
      </div>
      {filteredIntegrationCatalog.length === 0 ? (
        <div className="py-12 text-center">
          <Plug className={cn("mx-auto mb-2 h-6 w-6", settingsEmptyStateIconClassName)} />
          <p className={settingsEmptyStateTitleClassName}>
            {integrationCatalog.length === 0 ? "No integrations available." : "No integrations match your filter."}
          </p>
          {isIntegrationSurveyActive ? (
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">
              Can't find your integration?{" "}
              <button
                type="button"
                onClick={() => analytics.integrationRequested(organizationId)}
                className="text-blue-600 hover:underline dark:text-blue-400 font-medium"
              >
                Request it
              </button>
            </p>
          ) : null}
        </div>
      ) : (
        <div className="space-y-4">
          {filteredIntegrationCatalog.map((item) => {
            const connectedCount = item.instances.length;

            return (
              <div key={item.providerName} className={settingsPanelClassName}>
                <div className="p-4 flex items-start justify-between gap-4">
                  <div className="flex items-start gap-3">
                    <div className="mt-0.5 flex h-8 w-8 items-center justify-center">
                      <IntegrationIcon
                        integrationName={item.providerName}
                        iconSlug={item.integrationDef?.icon}
                        className="w-8 h-8 text-gray-500 dark:text-gray-400"
                      />
                    </div>
                    <div>
                      <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100">{item.providerLabel}</h3>
                      {item.integrationDef?.description ? (
                        <p className="mt-0.5 text-sm text-gray-800 dark:text-gray-400">
                          {item.integrationDef?.description}
                        </p>
                      ) : null}
                    </div>
                  </div>
                  {usesHostedGitHubAppInstall(item.integrationDef) ? (
                    <GitHubConnectControls
                      organizationId={organizationId}
                      definition={item.integrationDef}
                      canCreateIntegrations={canCreateIntegrations}
                      permissionsLoading={permissionsLoading}
                      onConnect={() => {
                        if (!item.integrationDef) return;
                        onConnectClick(item.integrationDef);
                      }}
                    />
                  ) : (
                    <PermissionTooltip
                      allowed={Boolean(item.integrationDef) && (canCreateIntegrations || permissionsLoading)}
                      message={
                        item.integrationDef
                          ? "You don't have permission to connect integrations."
                          : "This integration provider is no longer available for new connections."
                      }
                    >
                      <Button
                        variant="default"
                        size="sm"
                        onClick={() => {
                          if (!item.integrationDef) return;
                          onConnectClick(item.integrationDef);
                        }}
                        className="self-start"
                        disabled={!item.integrationDef || !canCreateIntegrations}
                      >
                        {item.integrationDef ? "Connect" : "Unavailable"}
                      </Button>
                    </PermissionTooltip>
                  )}
                </div>
                {item.instances.length > 0 ? (
                  <div className="pr-4 pb-4 pl-[60px]">
                    <p className="mb-2 text-xs text-gray-500 dark:text-gray-400">
                      {connectedCount} connected instance{connectedCount === 1 ? "" : "s"}
                    </p>
                    {item.instances.map((integration, index) => (
                      <IntegrationInstanceRow
                        key={integration.metadata?.id}
                        integration={integration}
                        index={index}
                        organizationId={organizationId}
                        canUpdateIntegrations={canUpdateIntegrations}
                        permissionsLoading={permissionsLoading}
                      />
                    ))}
                  </div>
                ) : null}
              </div>
            );
          })}
          {isIntegrationSurveyActive ? (
            <p className="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
              Can't find your integration?{" "}
              <button
                type="button"
                onClick={() => analytics.integrationRequested(organizationId)}
                className="text-blue-600 hover:underline dark:text-blue-400 font-medium"
              >
                Request it
              </button>
            </p>
          ) : null}
        </div>
      )}

      {isModalOpen && selectedIntegration ? (
        <IntegrationConnectModal
          selectedIntegration={selectedIntegration}
          selectedInstructions={selectedInstructions}
          integrationName={integrationName}
          configuration={configuration}
          organizationId={organizationId}
          canCreateIntegrations={canCreateIntegrations}
          isPending={createIntegrationMutation.isPending}
          isError={createIntegrationMutation.isError}
          error={createIntegrationMutation.error}
          createIntegrationNotice={createIntegrationNotice}
          onNameChange={onNameChange}
          onConfigurationChange={onConfigurationChange}
          onConnect={onConnect}
          onClose={onCloseModal}
        />
      ) : null}
    </div>
  );
}
