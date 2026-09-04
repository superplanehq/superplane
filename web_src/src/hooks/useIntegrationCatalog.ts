import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { usePermissions } from "@/contexts/usePermissions";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import type { IntegrationsIntegrationDefinition } from "@/api-client/types.gen";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";
import { analytics } from "@/lib/analytics";
import {
  isCapabilityBasedIntegrationDefinition,
  usesHostedGitHubAppInstall,
  usesPrivateGitHubAppWizard,
} from "@/lib/integrations";
import { connectPrivateGitHubApp } from "@/lib/privateGitHubApp";
import { posthog, isPostHogEnabled } from "@/posthog";
import { integrationDetailPath, integrationSetupPath, useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { buildIntegrationCatalog, filterIntegrationCatalog, integrationNameSet } from "@/lib/integrationCatalog";
import { persistGitHubSetupReturnPath, startDirectGitHubConnect } from "@/lib/startDirectGitHubConnect";
import { useMe } from "@/hooks/useMe";

const INTEGRATION_SURVEY_NAME = "Integration Survey";

export function useIntegrationCatalog(organizationId: string) {
  const navigate = useNavigate();
  const { data: me } = useMe();
  const integrationsBasePath = useIntegrationsBasePath(organizationId);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationDefinition | null>(null);
  const [integrationName, setIntegrationName] = useState("");
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [filterQuery, setFilterQuery] = useState("");
  const isIntegrationSurveyActive = useIntegrationSurveyActive();
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");

  const { data: availableIntegrations = [], isLoading: loadingAvailable } = useAvailableIntegrations();
  const { data: organizationIntegrations = [], isLoading: loadingInstalled } = useConnectedIntegrations(organizationId);
  const createIntegrationMutation = useCreateIntegration(organizationId, "integrations_page");
  const isLoading = loadingAvailable || loadingInstalled;
  const integrationNames = useMemo(() => integrationNameSet(organizationIntegrations), [organizationIntegrations]);
  const integrationCatalog = useMemo(
    () => buildIntegrationCatalog(availableIntegrations, organizationIntegrations),
    [availableIntegrations, organizationIntegrations],
  );
  const filteredIntegrationCatalog = useMemo(
    () => filterIntegrationCatalog(integrationCatalog, filterQuery),
    [filterQuery, integrationCatalog],
  );

  useReportPageReady(!isLoading && !permissionsLoading);

  const actions = useIntegrationCatalogActions({
    organizationId,
    integrationsBasePath,
    navigate,
    canCreateIntegrations,
    canUpdateIntegrations,
    currentUserId: me?.id,
    organizationIntegrations,
    integrationNames,
    selectedIntegration,
    setSelectedIntegration,
    integrationName,
    setIntegrationName,
    configuration,
    setConfiguration,
    setIsModalOpen,
    createIntegrationMutation,
  });

  return {
    organizationId,
    isLoading,
    permissionsLoading,
    canCreateIntegrations,
    canUpdateIntegrations,
    filterQuery,
    setFilterQuery,
    isIntegrationSurveyActive,
    integrationCatalog,
    filteredIntegrationCatalog,
    selectedIntegration,
    selectedInstructions: selectedIntegration?.instructions?.trim() ?? "",
    integrationName,
    setIntegrationName,
    configuration,
    setConfiguration,
    isModalOpen,
    createIntegrationMutation,
    createIntegrationNotice: createIntegrationMutation.isError
      ? getUsageLimitNotice(createIntegrationMutation.error, organizationId)
      : null,
    handlePrivateAppClick: (definition?: IntegrationsIntegrationDefinition) =>
      startCatalogPrivateGitHubApp({
        organizationId,
        integrationsBasePath,
        navigate,
        integrationNames,
        organizationIntegrations: organizationIntegrations ?? [],
        currentUserId: me?.id,
        createIntegrationMutation,
        definition,
      }),
    ...actions,
  };
}

function useIntegrationSurveyActive() {
  const [isActive, setIsActive] = useState(false);

  useEffect(() => {
    if (!isPostHogEnabled) {
      return;
    }
    posthog.getSurveys((surveys) => {
      setIsActive(
        surveys.some(
          (survey) => survey.name === INTEGRATION_SURVEY_NAME && survey.start_date != null && survey.end_date == null,
        ),
      );
    }, true);
  }, []);

  return isActive;
}

function useIntegrationCatalogActions({
  organizationId,
  integrationsBasePath,
  navigate,
  canCreateIntegrations,
  canUpdateIntegrations,
  currentUserId,
  organizationIntegrations,
  integrationNames,
  selectedIntegration,
  setSelectedIntegration,
  integrationName,
  setIntegrationName,
  configuration,
  setConfiguration,
  setIsModalOpen,
  createIntegrationMutation,
}: {
  organizationId: string;
  integrationsBasePath: string;
  navigate: ReturnType<typeof useNavigate>;
  canCreateIntegrations: boolean;
  canUpdateIntegrations: boolean;
  currentUserId?: string;
  organizationIntegrations: ReturnType<typeof useConnectedIntegrations>["data"];
  integrationNames: Set<string>;
  selectedIntegration: IntegrationsIntegrationDefinition | null;
  setSelectedIntegration: (integration: IntegrationsIntegrationDefinition | null) => void;
  integrationName: string;
  setIntegrationName: (name: string) => void;
  configuration: Record<string, unknown>;
  setConfiguration: (configuration: Record<string, unknown>) => void;
  setIsModalOpen: (open: boolean) => void;
  createIntegrationMutation: ReturnType<typeof useCreateIntegration>;
}) {
  return {
    handleConnectClick: (integration: IntegrationsIntegrationDefinition) => {
      if (!canCreateIntegrations) {
        return;
      }
      if (usesHostedGitHubAppInstall(integration)) {
        analytics.integrationConnectStart("github", "integrations_page", organizationId);
        void startDirectGitHubConnect({
          organizationId,
          returnTo: integrationsBasePath,
          integrationsBasePath,
          existingNames: integrationNames,
          connected: organizationIntegrations ?? [],
          currentUserId,
          goTo: navigate,
          create: async (payload) => {
            const response = await createIntegrationMutation.mutateAsync(payload);
            return response.data;
          },
          update: persistGitHubSetupReturnPath(organizationId),
        }).catch((error) => {
          showErrorToast(getUsageLimitToastMessage(error, "Failed to connect GitHub"));
        });
        return;
      }
      if (isCapabilityBasedIntegrationDefinition(integration)) {
        if (!integration.name) {
          return;
        }
        analytics.integrationConnectStart(integration.name, "integrations_page", organizationId);
        navigate(integrationSetupPath(integrationsBasePath, integration.name));
        return;
      }
      setSelectedIntegration(integration);
      setIntegrationName(getNextIntegrationName(integration.name || "integration", integrationNames));
      setConfiguration({});
      setIsModalOpen(true);
      analytics.integrationConnectStart(integration.name ?? "", "integrations_page", organizationId);
    },
    handleConnect: async () => {
      if (!canCreateIntegrations || !selectedIntegration?.name) {
        return;
      }
      try {
        const result = await createIntegrationMutation.mutateAsync({
          integrationName: selectedIntegration.name,
          name: integrationName,
          configuration,
        });
        setIsModalOpen(false);
        setSelectedIntegration(null);
        setIntegrationName("");
        setConfiguration({});
        const createdId = result.data?.integration?.metadata?.id;
        if (createdId) {
          navigate(integrationDetailPath(integrationsBasePath, createdId));
        }
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to create integration"));
      }
    },
    handleRequestIntegration: () => {
      analytics.integrationRequested(organizationId);
    },
    handleCloseModal: () => {
      setIsModalOpen(false);
      setSelectedIntegration(null);
      setIntegrationName("");
      setConfiguration({});
      createIntegrationMutation.reset();
    },
    openInstance: (providerName: string | undefined, integrationId: string | undefined, inSetup: boolean) => {
      if (!canUpdateIntegrations) {
        return;
      }
      if (providerName && inSetup) {
        navigate(integrationSetupPath(integrationsBasePath, providerName), { state: { integrationId } });
        return;
      }
      if (integrationId) {
        navigate(integrationDetailPath(integrationsBasePath, integrationId), { state: { tab: "configuration" } });
      }
    },
  };
}

function startCatalogPrivateGitHubApp({
  organizationId,
  integrationsBasePath,
  navigate,
  integrationNames,
  organizationIntegrations,
  currentUserId,
  createIntegrationMutation,
  definition,
}: {
  organizationId: string;
  integrationsBasePath: string;
  navigate: ReturnType<typeof useNavigate>;
  integrationNames: Set<string>;
  organizationIntegrations: NonNullable<ReturnType<typeof useConnectedIntegrations>["data"]>;
  currentUserId?: string;
  createIntegrationMutation: ReturnType<typeof useCreateIntegration>;
  definition?: IntegrationsIntegrationDefinition;
}) {
  void connectPrivateGitHubApp({
    useWizard: usesPrivateGitHubAppWizard(definition),
    organizationId,
    returnTo: integrationsBasePath,
    integrationsBasePath,
    existingNames: integrationNames,
    connected: organizationIntegrations,
    currentUserId,
    goTo: navigate,
    create: async (payload) => {
      const response = await createIntegrationMutation.mutateAsync(payload);
      return response.data;
    },
  }).catch((error) => {
    showErrorToast(getUsageLimitToastMessage(error, "Failed to connect GitHub"));
  });
}
