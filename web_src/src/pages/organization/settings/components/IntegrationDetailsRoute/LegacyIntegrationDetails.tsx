import type { ConfigurationField, OrganizationsIntegration } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useAvailableIntegrations, useDeleteIntegration, useUpdateIntegration } from "@/hooks/useIntegrations";
import { useIntegrationConfigureOpen } from "@/lib/analytics";
import { followBrowserAction } from "@/lib/browserAction";
import { getApiErrorMessage } from "@/lib/errors";
import {
  hostedGitHubAppSlug,
  hostedGitHubInstallRequested,
  hostedGitHubState,
  pendingGitHubInstallations,
} from "@/lib/hostedGitHubInstall";
import { GITHUB_SETUP_REQUEST_PARAM, GITHUB_SETUP_REQUEST_VALUE } from "@/lib/integrationSetupReturn";
import { useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { cn } from "@/lib/utils";
import { HostedGitHubInstallPicker } from "@/pages/organization/settings/components/HostedGitHubInstallPicker";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { CircleX, Copy } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { DeleteModal } from "../CapabilityBasedIntegrationDetails/DeleteModal";
import { Header } from "../CapabilityBasedIntegrationDetails/Header";
import { UsageTab } from "../CapabilityBasedIntegrationDetails/UsageTab";
import { getActiveTabClass, groupNodeRefsByCanvas } from "../CapabilityBasedIntegrationDetails/lib";
import {
  configurationSubmitPayload,
  editableConfigurationField,
  editableConfigurationValue,
  nextConfigurationValue,
} from "./legacyConfiguration";

interface LegacyIntegrationDetailsProps {
  organizationId: string;
  integration: OrganizationsIntegration;
}

type LegacyIntegrationTab = "configuration" | "usage";

function TabButton({
  activeTab,
  value,
  children,
  onSelect,
}: {
  activeTab: LegacyIntegrationTab;
  value: LegacyIntegrationTab;
  children: React.ReactNode;
  onSelect: (value: LegacyIntegrationTab) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(value)}
      className={cn(
        "mr-4 mb-[-1px] border-b py-2 text-sm font-medium transition-colors",
        getActiveTabClass(activeTab === value),
      )}
    >
      {children}
    </button>
  );
}

export function LegacyIntegrationDetails({ organizationId, integration }: LegacyIntegrationDetailsProps) {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const integrationsHref = useIntegrationsBasePath(organizationId);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [activeTab, setActiveTab] = useState<LegacyIntegrationTab>("configuration");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [savedConfiguration, setSavedConfiguration] = useState<Record<string, unknown>>({});
  const [integrationName, setIntegrationName] = useState("");
  const [savedIntegrationName, setSavedIntegrationName] = useState("");

  const canUpdateIntegrations = canAct("integrations", "update");
  const canDeleteIntegrations = canAct("integrations", "delete");
  const integrationId = integration.metadata?.id || "";
  const providerName = integration.metadata?.integrationName || "";
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = availableIntegrations.find((definition) => definition.name === providerName);
  const updateMutation = useUpdateIntegration(organizationId, integrationId);
  const deleteMutation = useDeleteIntegration(organizationId, integrationId);

  usePageTitle(["Integrations", integration.metadata?.name]);
  useIntegrationConfigureOpen(integration, integrationId, "integrations_page", organizationId);

  useEffect(() => {
    const nextConfiguration = integration.spec?.configuration || {};
    setConfiguration(nextConfiguration);
    setSavedConfiguration(nextConfiguration);
  }, [integration.spec?.configuration]);

  useEffect(() => {
    const nextName = integration.metadata?.name || providerName;
    setIntegrationName(nextName);
    setSavedIntegrationName(nextName);
  }, [integration.metadata?.name, providerName]);

  const workflowGroups = useMemo(
    () => groupNodeRefsByCanvas(integration.status?.usedIn || []),
    [integration.status?.usedIn],
  );
  const pendingInstallations = pendingGitHubInstallations(integration.status?.metadata);
  const pendingInstallState = hostedGitHubState(integration.status?.metadata);
  const installRequested =
    searchParams.get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE ||
    hostedGitHubInstallRequested(integration.status?.metadata);
  const showInstallPicker =
    pendingInstallations.length >= 2 && pendingInstallState !== "" && integration.status?.state !== "ready";
  const browserAction = integration.status?.browserAction;
  const instructions = integrationDef?.instructions?.trim();
  const hasChanges =
    integrationName.trim() !== savedIntegrationName ||
    JSON.stringify(configuration) !== JSON.stringify(savedConfiguration);

  const handleConfigSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const name = integrationName.trim();
    if (!canUpdateIntegrations || !hasChanges) return;
    if (!name) {
      showErrorToast("Integration name is required");
      return;
    }

    try {
      await updateMutation.mutateAsync({
        name,
        configuration: configurationSubmitPayload(integrationDef?.configuration, configuration),
      });
      setSavedIntegrationName(name);
      setSavedConfiguration(configuration);
      showSuccessToast("Integration saved");
    } catch (error) {
      showErrorToast(`SuperPlane could not save the integration: ${getApiErrorMessage(error)}`);
    }
  };

  const handleDelete = async () => {
    if (!canDeleteIntegrations) return;
    try {
      await deleteMutation.mutateAsync({ integrationName: providerName });
      navigate(integrationsHref);
    } catch {
      showErrorToast("SuperPlane could not delete the integration. Try again.");
    }
  };

  return (
    <div className="mx-auto w-full max-w-3xl px-3 pt-14 pb-10">
      <Header
        organizationId={organizationId}
        integration={integration}
        integrationDef={integrationDef}
        canDeleteIntegrations={canDeleteIntegrations}
        permissionsLoading={permissionsLoading}
        onRequestDelete={() => setShowDeleteConfirm(true)}
      />

      <div className="space-y-6">
        {installRequested && integration.status?.state !== "ready" ? (
          <Alert data-testid="github-install-requested">
            <AlertTitle>Waiting for GitHub approval</AlertTitle>
            <AlertDescription>
              <p>Ask a GitHub organization admin to approve the SuperPlane GitHub App.</p>
              <p>After they approve, click Connect GitHub again.</p>
            </AlertDescription>
          </Alert>
        ) : null}

        {integration.status?.state === "error" && integration.status.stateDescription ? (
          <Alert className="border-destructive/40 bg-destructive/10 text-destructive [&>svg+div]:translate-y-0 [&>svg]:top-[14px] [&>svg]:text-destructive">
            <CircleX className="size-4" />
            <AlertTitle>Connection issue</AlertTitle>
            <AlertDescription>{integration.status.stateDescription}</AlertDescription>
          </Alert>
        ) : null}

        {showInstallPicker ? (
          <HostedGitHubInstallPicker
            installations={pendingInstallations}
            state={pendingInstallState}
            appSlug={hostedGitHubAppSlug(integration.status?.metadata)}
          />
        ) : browserAction ? (
          <IntegrationInstructions
            description={browserAction.description}
            tone="settings"
            onContinue={browserAction.url ? () => followBrowserAction(browserAction) : undefined}
          />
        ) : null}

        {instructions ? <IntegrationInstructions description={instructions} tone="settings" /> : null}

        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as LegacyIntegrationTab)}
          className="w-full"
        >
          <div className="border-b border-border">
            <div className="flex flex-wrap px-4">
              <TabButton activeTab={activeTab} value="configuration" onSelect={setActiveTab}>
                Configuration
              </TabButton>
              <TabButton activeTab={activeTab} value="usage" onSelect={setActiveTab}>
                Usage
              </TabButton>
            </div>
          </div>

          <TabsContent value="configuration" className="mt-4">
            <section className="overflow-hidden rounded-lg border border-border bg-card text-card-foreground">
              <div className="border-b border-border px-6 py-5">
                <h2 className="text-base font-medium text-foreground">Configuration</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  Update the integration name or replace its stored credentials.
                </p>
              </div>

              <PermissionTooltip
                allowed={canUpdateIntegrations || permissionsLoading}
                message="You do not have permission to update integrations."
                className="w-full"
              >
                <form onSubmit={handleConfigSubmit}>
                  <div className="max-w-2xl space-y-5 px-6 py-6">
                    <div className="space-y-2">
                      <Label htmlFor="legacy-integration-name">
                        Integration name <span aria-hidden>*</span>
                      </Label>
                      <Input
                        id="legacy-integration-name"
                        value={integrationName}
                        onChange={(event) => setIntegrationName(event.target.value)}
                        disabled={!canUpdateIntegrations}
                        autoComplete="off"
                      />
                      <p className="text-xs text-muted-foreground">
                        Use a unique name that identifies this connection.
                      </p>
                    </div>

                    {integrationDef?.configuration?.length ? (
                      integrationDef.configuration.map((field: ConfigurationField) => {
                        if (!field.name) return null;
                        const storedValue = configuration[field.name];
                        return (
                          <ConfigurationFieldRenderer
                            key={field.name}
                            field={editableConfigurationField(field, storedValue)}
                            value={editableConfigurationValue(field, storedValue)}
                            onChange={(value) =>
                              setConfiguration((current) => ({
                                ...current,
                                [field.name!]: nextConfigurationValue(field, current[field.name!], value),
                              }))
                            }
                            allValues={configuration}
                            organizationId={organizationId}
                            integrationId={integrationId}
                          />
                        );
                      })
                    ) : (
                      <p className="text-sm text-muted-foreground">This integration has no configuration fields.</p>
                    )}
                  </div>

                  <div className="flex flex-wrap items-center gap-3 border-t border-border bg-muted/40 px-6 py-4">
                    <LoadingButton
                      type="submit"
                      color="blue"
                      disabled={!integrationName.trim() || !canUpdateIntegrations || !hasChanges}
                      loading={updateMutation.isPending}
                      loadingText="Saving..."
                    >
                      Save changes
                    </LoadingButton>
                    <Button
                      type="button"
                      variant="outline"
                      disabled={!hasChanges || updateMutation.isPending}
                      onClick={() => {
                        setIntegrationName(savedIntegrationName);
                        setConfiguration(savedConfiguration);
                      }}
                    >
                      Reset
                    </Button>
                    {updateMutation.isError ? (
                      <p className="text-sm text-destructive">
                        SuperPlane could not save the integration. {getApiErrorMessage(updateMutation.error)}
                      </p>
                    ) : null}
                  </div>
                </form>
              </PermissionTooltip>
            </section>

            {typeof integration.status?.metadata?.webhookUrl === "string" ? (
              <section className="mt-4 rounded-lg border border-border bg-card p-6 text-card-foreground">
                <h2 className="text-base font-medium text-foreground">Webhook URL</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  Add this URL to the external service that sends webhook events.
                </p>
                <div className="mt-4 flex max-w-2xl items-center gap-2">
                  <Input value={integration.status.metadata.webhookUrl} readOnly className="font-mono text-sm" />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={async () => {
                      await navigator.clipboard.writeText(integration.status!.metadata!.webhookUrl as string);
                      showSuccessToast("Webhook URL copied");
                    }}
                  >
                    <Copy className="size-4" aria-hidden />
                    Copy
                  </Button>
                </div>
              </section>
            ) : null}
          </TabsContent>

          <TabsContent value="usage" className="mt-4">
            <UsageTab organizationId={organizationId} workflowGroups={workflowGroups} />
          </TabsContent>
        </Tabs>
      </div>

      <DeleteModal
        open={showDeleteConfirm}
        integrationName={integration.metadata?.name}
        canDeleteIntegrations={canDeleteIntegrations}
        isDeleting={deleteMutation.isPending}
        hasDeleteError={deleteMutation.isError}
        onDelete={handleDelete}
        onClose={() => setShowDeleteConfirm(false)}
      />
    </div>
  );
}
