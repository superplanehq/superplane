import { ArrowLeft, ExternalLink, Loader2, Trash2 } from "lucide-react";
import { useNavigate, useParams, useLocation } from "react-router-dom";
import { useState, useEffect, useMemo } from "react";
import ReactMarkdown from "react-markdown";
import {
  useAvailableIntegrations,
  useDeleteIntegration,
  useIntegration,
  useUpdateIntegration,
} from "@/hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/ui/tabs";
import { Alert, AlertDescription } from "@/ui/alert";
import { resolveIcon } from "@/lib/utils";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import daytonaIcon from "@/assets/icons/integrations/daytona.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import smtpIcon from "@/assets/icons/integrations/smtp.svg";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

interface IntegrationDetailsProps {
  organizationId: string;
}

export function IntegrationDetails({ organizationId }: IntegrationDetailsProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { integrationId } = useParams<{ integrationId: string }>();
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const { data: integration, isLoading, error } = useIntegration(organizationId, integrationId || "");

  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = integration
    ? availableIntegrations.find((i) => i.name === integration.spec?.integrationName)
    : undefined;
  const appLogoMap: Record<string, string> = {
    aws: awsIcon,
    dash0: dash0Icon,
    github: githubIcon,
    openai: openAiIcon,
    daytona: daytonaIcon,
    "open-ai": openAiIcon,
    pagerduty: pagerDutyIcon,
    semaphore: SemaphoreLogo,
    slack: slackIcon,
    smtp: smtpIcon,
  };

  const renderAppIcon = (slug: string | undefined, appName: string | undefined, className: string) => {
    const logo = appName ? appLogoMap[appName] : undefined;
    if (logo) {
      return (
        <span className={className}>
          <img src={logo} alt="" className="h-full w-full object-contain" />
        </span>
      );
    }
    const IconComponent = resolveIcon(slug);
    return <IconComponent className={className} />;
  };

  const updateMutation = useUpdateIntegration(organizationId, integrationId || "");
  const deleteMutation = useDeleteIntegration(organizationId, integrationId || "");

  // Initialize config values when installation loads
  useEffect(() => {
    if (integration?.spec?.configuration) {
      setConfigValues(integration.spec.configuration);
    }
  }, [integration]);

  // Group usedIn nodes by workflow
  const workflowGroups = useMemo(() => {
    if (!integration?.status?.usedIn) return [];

    const groups = new Map<string, { workflowName: string; nodes: Array<{ nodeId: string; nodeName: string }> }>();
    integration.status.usedIn.forEach((nodeRef) => {
      const workflowId = nodeRef.workflowId || "";
      const workflowName = nodeRef.workflowName || workflowId;
      const nodeId = nodeRef.nodeId || "";
      const nodeName = nodeRef.nodeName || nodeId;

      if (!groups.has(workflowId)) {
        groups.set(workflowId, { workflowName, nodes: [] });
      }
      groups.get(workflowId)?.nodes.push({ nodeId, nodeName });
    });

    return Array.from(groups.entries()).map(([workflowId, data]) => ({
      workflowId,
      workflowName: data.workflowName,
      nodes: data.nodes,
    }));
  }, [integration?.status?.usedIn]);

  const handleConfigSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await updateMutation.mutateAsync(configValues);
      navigate(`/${organizationId}/settings/integrations`);
    } catch (error) {
      console.error("Failed to update configuration:", error);
    }
  };

  const handleBrowserAction = () => {
    if (!integration?.status?.browserAction) return;

    const { url, method, formFields } = integration.status.browserAction;

    if (method?.toUpperCase() === "POST" && formFields) {
      // Create a hidden form and submit it
      const form = document.createElement("form");
      form.method = "POST";
      form.action = url || "";
      form.target = "_blank";
      form.style.display = "none";

      // Add form fields
      Object.entries(formFields).forEach(([key, value]) => {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = key;
        input.value = String(value);
        form.appendChild(input);
      });

      document.body.appendChild(form);
      form.submit();
      document.body.removeChild(form);
    } else {
      // For GET requests or no form fields, just open the URL
      if (url) {
        window.open(url, "_blank");
      }
    }
  };

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync();
      navigate(`/${organizationId}/settings/integrations`);
    } catch (error) {
      console.error("Failed to delete integration:", error);
    }
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={() => navigate(`/${organizationId}/settings/integrations`)}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="flex justify-center items-center h-32">
          <Loader2 className="w-8 h-8 animate-spin text-gray-500 dark:text-gray-400" />
        </div>
      </div>
    );
  }

  if (error || !integration) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={() => navigate(`/${organizationId}/settings/integrations`)}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <p className="text-gray-500 dark:text-gray-400">Integration not found</p>
        </div>
      </div>
    );
  }

  const defaultTab = location.state?.tab === "configuration" ? "configuration" : "overview";

  return (
    <div className="pt-6">
      <div className="flex items-center gap-4 mb-6">
        <button
          onClick={() => navigate(`/${organizationId}/settings/integrations`)}
          className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        {integrationDef?.icon &&
          renderAppIcon(integrationDef.icon, integrationDef.name || integration?.spec?.integrationName, "w-6 h-6")}
        <div className="flex-1">
          <h4 className="text-2xl font-semibold">{integration.metadata?.name || integration.spec?.integrationName}</h4>
          {integration.spec?.integrationName && integration.metadata?.name !== integration.spec?.integrationName && (
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Integration: {integration.spec.integrationName}
            </p>
          )}
        </div>
      </div>

      <Tabs defaultValue={defaultTab} className="w-full">
        <TabsList className="mb-6">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              <h2 className="text-lg font-medium mb-4">Integration Details</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Integration ID</h3>
                  <p className="text-sm text-gray-800 dark:text-gray-100 font-mono">{integration.metadata?.id}</p>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">State</h3>
                  <span
                    className={`inline-flex px-2 py-0.5 text-xs font-medium rounded ${
                      integration.status?.state === "ready"
                        ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                        : integration.status?.state === "error"
                          ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                          : "bg-orange-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
                    }`}
                  >
                    {(integration.status?.state || "unknown").charAt(0).toUpperCase() +
                      (integration.status?.state || "unknown").slice(1)}
                  </span>
                  {integration.status?.stateDescription && (
                    <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                      {integration.status.stateDescription}
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Used By */}
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              <h2 className="text-lg font-medium mb-4">Used By</h2>
              {workflowGroups.length > 0 ? (
                <>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">
                    This integration is currently used in the following canvases:
                  </p>
                  <div className="space-y-2">
                    {workflowGroups.map((group) => (
                      <button
                        key={group.workflowId}
                        onClick={() => window.open(`/${organizationId}/workflows/${group.workflowId}`, "_blank")}
                        className="w-full flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-md border border-gray-300 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors text-left"
                      >
                        <div className="flex-1">
                          <p className="text-sm font-medium text-gray-800 dark:text-gray-100">
                            Canvas: {group.workflowName}
                          </p>
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Used in {group.nodes.length} node{group.nodes.length !== 1 ? "s" : ""}:{" "}
                            {group.nodes.map((node) => node.nodeName).join(", ")}
                          </p>
                        </div>
                        <ExternalLink className="w-4 h-4 text-gray-400 dark:text-gray-500 shrink-0" />
                      </button>
                    ))}
                  </div>
                </>
              ) : (
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  This integration is not used in any workflow yet.
                </p>
              )}
            </div>
          </div>

          {/* Danger Zone */}
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-red-200 dark:border-red-800">
            <div className="p-6">
              <h2 className="text-lg font-medium text-red-600 dark:text-red-400 mb-2">Danger Zone</h2>
              <p className="text-sm text-gray-800 dark:text-gray-100 mb-4">
                Once you delete this integration, all its data will be permanently deleted. This action cannot be
                undone.
              </p>
              <Button
                variant="outline"
                onClick={() => setShowDeleteConfirm(true)}
                className="border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 hover:text-red-600 dark:hover:text-red-400 gap-1"
              >
                <Trash2 className="w-4 h-4" />
                Delete Integration
              </Button>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="configuration">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              {integration?.status?.browserAction && (
                <Alert className="mb-6 bg-orange-100 dark:bg-yellow-900/20 border-orange-200 dark:border-yellow-800">
                  <div className="flex items-start justify-between gap-4">
                    <AlertDescription className="flex-1 text-yellow-800 dark:text-yellow-200 [&_ol]:list-decimal [&_ol]:ml-6 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-6 [&_ul]:space-y-1">
                      {integration.status.browserAction.description && (
                        <ReactMarkdown>{integration.status.browserAction.description}</ReactMarkdown>
                      )}
                    </AlertDescription>
                    {integration.status.browserAction.url && (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={handleBrowserAction}
                        className="shrink-0 px-3 py-1.5"
                      >
                        <ExternalLink className="w-4 h-4 mr-2" />
                        Continue
                      </Button>
                    )}
                  </div>
                </Alert>
              )}

              {integrationDef?.configuration && integrationDef.configuration.length > 0 ? (
                <form onSubmit={handleConfigSubmit} className="space-y-4">
                  {integrationDef.configuration.map((field) => (
                    <ConfigurationFieldRenderer
                      key={field.name}
                      field={field}
                      value={configValues[field.name!]}
                      onChange={(value) => setConfigValues({ ...configValues, [field.name!]: value })}
                      allValues={configValues}
                      domainId={organizationId}
                      domainType="DOMAIN_TYPE_ORGANIZATION"
                      organizationId={organizationId}
                      appInstallationId={integration?.metadata?.id}
                    />
                  ))}

                  <div className="flex items-center gap-3 pt-4">
                    <Button type="submit" color="blue" disabled={updateMutation.isPending}>
                      {updateMutation.isPending ? (
                        <>
                          <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                          Saving...
                        </>
                      ) : (
                        "Save Configuration"
                      )}
                    </Button>
                    {updateMutation.isSuccess && (
                      <span className="text-sm text-green-600 dark:text-green-400">
                        Configuration updated successfully!
                      </span>
                    )}
                    {updateMutation.isError && (
                      <span className="text-sm text-red-600 dark:text-red-400">Failed to update configuration</span>
                    )}
                  </div>
                </form>
              ) : (
                <p className="text-sm text-gray-500 dark:text-gray-400">No configuration fields available.</p>
              )}
            </div>
          </div>
        </TabsContent>
      </Tabs>

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-2">
                Delete {integration?.metadata?.name || "integration"}?
              </h3>
              <p className="text-sm text-gray-800 dark:text-gray-100 mb-6">
                This cannot be undone. All data will be permanently deleted.
              </p>
              <div className="flex justify-start gap-3">
                <Button
                  color="blue"
                  onClick={handleDelete}
                  disabled={deleteMutation.isPending}
                  className="bg-red-600 hover:bg-red-700 dark:bg-red-600 dark:hover:bg-red-700"
                >
                  {deleteMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Deleting...
                    </>
                  ) : (
                    "Delete"
                  )}
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setShowDeleteConfirm(false)}
                  disabled={deleteMutation.isPending}
                >
                  Cancel
                </Button>
              </div>
              {deleteMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to delete integration. Please try again.
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
