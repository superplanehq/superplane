import { ArrowLeft, ExternalLink, Loader2, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { useState, useEffect, useMemo } from "react";
import ReactMarkdown from "react-markdown";
import {
  useApplicationInstallation,
  useAvailableApplications,
  useUpdateApplication,
  useUninstallApplication,
} from "@/hooks/useApplications";
import { Button } from "@/components/ui/button";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/ui/tabs";
import { Alert, AlertDescription } from "@/ui/alert";
import { resolveIcon } from "@/lib/utils";
import githubIcon from "@/assets/icons/integrations/github.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

interface ApplicationDetailsProps {
  organizationId: string;
}

export function ApplicationDetails({ organizationId }: ApplicationDetailsProps) {
  const navigate = useNavigate();
  const { installationId } = useParams<{ installationId: string }>();
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const { data: installation, isLoading, error } = useApplicationInstallation(organizationId, installationId || "");

  const { data: availableApps = [] } = useAvailableApplications();
  const appDefinition = installation ? availableApps.find((app) => app.name === installation.spec?.appName) : undefined;
  const appLogoMap: Record<string, string> = {
    github: githubIcon,
    semaphore: SemaphoreLogo,
    pagerduty: pagerDutyIcon,
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

  const updateMutation = useUpdateApplication(organizationId, installationId || "");
  const uninstallMutation = useUninstallApplication(organizationId, installationId || "");

  // Initialize config values when installation loads
  useEffect(() => {
    if (installation?.spec?.configuration) {
      setConfigValues(installation.spec.configuration);
    }
  }, [installation]);

  // Group usedIn nodes by workflow
  const workflowGroups = useMemo(() => {
    if (!installation?.status?.usedIn) return [];

    const groups = new Map<string, { workflowName: string; nodes: Array<{ nodeId: string; nodeName: string }> }>();
    installation.status.usedIn.forEach((nodeRef) => {
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
  }, [installation?.status?.usedIn]);

  const handleConfigSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await updateMutation.mutateAsync(configValues);
      navigate(`/${organizationId}/settings/applications`);
    } catch (error) {
      console.error("Failed to update configuration:", error);
    }
  };

  const handleBrowserAction = () => {
    if (!installation?.status?.browserAction) return;

    const { url, method, formFields } = installation.status.browserAction;

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

  const handleUninstall = async () => {
    try {
      await uninstallMutation.mutateAsync();
      navigate(`/${organizationId}/settings/applications`);
    } catch (error) {
      console.error("Failed to uninstall application:", error);
    }
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={() => navigate(`/${organizationId}/settings/applications`)}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h4 className="text-2xl font-semibold">Application Details</h4>
        </div>
        <div className="flex justify-center items-center h-32">
          <Loader2 className="w-8 h-8 animate-spin text-gray-500 dark:text-gray-400" />
        </div>
      </div>
    );
  }

  if (error || !installation) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={() => navigate(`/${organizationId}/settings/applications`)}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h4 className="text-2xl font-semibold">Application Details</h4>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <p className="text-gray-500 dark:text-gray-400">Application installation not found</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      <div className="flex items-center gap-4 mb-6">
        <button
          onClick={() => navigate(`/${organizationId}/settings/applications`)}
          className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        {appDefinition?.icon &&
          renderAppIcon(appDefinition.icon, appDefinition.name || installation?.spec?.appName, "w-6 h-6")}
        <div className="flex-1">
          <h4 className="text-2xl font-semibold">{installation.metadata?.name || installation.spec?.appName}</h4>
          {installation.spec?.appName && installation.metadata?.name !== installation.spec?.appName && (
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">Application: {installation.spec.appName}</p>
          )}
        </div>
      </div>

      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="mb-6">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              <h2 className="text-lg font-medium mb-4">Installation Details</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Installation ID</h3>
                  <p className="text-sm text-gray-800 dark:text-gray-100 font-mono">{installation.metadata?.id}</p>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">State</h3>
                  <span
                    className={`inline-flex px-2 py-0.5 text-xs font-medium rounded ${
                      installation.status?.state === "ready"
                        ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                        : installation.status?.state === "error"
                          ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                          : "bg-orange-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
                    }`}
                  >
                    {(installation.status?.state || "unknown").charAt(0).toUpperCase() +
                      (installation.status?.state || "unknown").slice(1)}
                  </span>
                  {installation.status?.stateDescription && (
                    <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                      {installation.status.stateDescription}
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
                    This app installation is currently used in the following canvases:
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
                  This app installation is not used in any workflow yet.
                </p>
              )}
            </div>
          </div>

          {/* Danger Zone */}
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-red-200 dark:border-red-800">
            <div className="p-6">
              <h2 className="text-lg font-medium text-red-600 dark:text-red-400 mb-2">Danger Zone</h2>
              <p className="text-sm text-gray-800 dark:text-gray-100 mb-4">
                Once you uninstall this application, all its data will be permanently deleted. This action cannot be
                undone.
              </p>
              <Button
                variant="outline"
                onClick={() => setShowDeleteConfirm(true)}
                className="border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 hover:text-red-600 dark:hover:text-red-400 gap-1"
              >
                <Trash2 className="w-4 h-4" />
                Uninstall Application
              </Button>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="configuration">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              {installation?.status?.browserAction && (
                <Alert className="mb-6 bg-orange-100 dark:bg-yellow-900/20 border-orange-200 dark:border-yellow-800">
                  <div className="flex items-start justify-between gap-4">
                    <AlertDescription className="flex-1 text-yellow-800 dark:text-yellow-200 [&_ol]:list-decimal [&_ol]:ml-6 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-6 [&_ul]:space-y-1">
                      {installation.status.browserAction.description && (
                        <ReactMarkdown>{installation.status.browserAction.description}</ReactMarkdown>
                      )}
                    </AlertDescription>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleBrowserAction}
                      className="shrink-0 px-3 py-1.5"
                    >
                      <ExternalLink className="w-4 h-4 mr-2" />
                      Continue
                    </Button>
                  </div>
                </Alert>
              )}

              {appDefinition?.configuration && appDefinition.configuration.length > 0 ? (
                <form onSubmit={handleConfigSubmit} className="space-y-4">
                  {appDefinition.configuration.map((field) => (
                    <ConfigurationFieldRenderer
                      key={field.name}
                      field={field}
                      value={configValues[field.name!]}
                      onChange={(value) => setConfigValues({ ...configValues, [field.name!]: value })}
                      allValues={configValues}
                      domainId={organizationId}
                      domainType="DOMAIN_TYPE_ORGANIZATION"
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
                Uninstall {installation?.metadata?.name || "application"}?
              </h3>
              <p className="text-sm text-gray-800 dark:text-gray-100 mb-6">
                This cannot be undone. All data will be permanently deleted.
              </p>
              <div className="flex justify-start gap-3">
                <Button
                  color="blue"
                  onClick={handleUninstall}
                  disabled={uninstallMutation.isPending}
                  className="bg-red-600 hover:bg-red-700 dark:bg-red-600 dark:hover:bg-red-700"
                >
                  {uninstallMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Uninstalling...
                    </>
                  ) : (
                    "Uninstall"
                  )}
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setShowDeleteConfirm(false)}
                  disabled={uninstallMutation.isPending}
                >
                  Cancel
                </Button>
              </div>
              {uninstallMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to uninstall application. Please try again.
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
