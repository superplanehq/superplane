import { ArrowLeft, ExternalLink, Loader2 } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { useState, useEffect } from "react";
import { useApplicationInstallation, useAvailableApplications, useUpdateApplication } from "@/hooks/useApplications";
import { Button } from "@/ui/button";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";

interface ApplicationDetailsProps {
  organizationId: string;
}

type Tab = "overview" | "configuration";

export function ApplicationDetails({ organizationId }: ApplicationDetailsProps) {
  const navigate = useNavigate();
  const { installationId } = useParams<{ installationId: string }>();
  const [activeTab, setActiveTab] = useState<Tab>("overview");
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});

  const { data: installation, isLoading, error } = useApplicationInstallation(organizationId, installationId || "");

  const { data: availableApps = [] } = useAvailableApplications();
  const appDefinition = installation ? availableApps.find((app) => app.name === installation.appName) : undefined;

  const updateMutation = useUpdateApplication(organizationId, installationId || "");

  // Initialize config values when installation loads
  useEffect(() => {
    if (installation?.configuration) {
      setConfigValues(installation.configuration);
    }
  }, [installation]);

  const handleConfigSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await updateMutation.mutateAsync(configValues);
    } catch (error) {
      console.error("Failed to update configuration:", error);
    }
  };

  const handleBrowserAction = () => {
    if (!installation?.browserAction) return;

    const { url, method, formFields } = installation.browserAction;

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

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={() => navigate(`/${organizationId}/settings/applications`)}
            className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h4 className="text-2xl font-semibold">Application Details</h4>
        </div>
        <div className="flex justify-center items-center h-32">
          <Loader2 className="w-8 h-8 animate-spin text-zinc-500 dark:text-zinc-400" />
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
            className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h4 className="text-2xl font-semibold">Application Details</h4>
        </div>
        <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
          <p className="text-zinc-600 dark:text-zinc-400">Application installation not found</p>
        </div>
      </div>
    );
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: "overview", label: "Overview" },
    { id: "configuration", label: "Configuration" },
  ];

  return (
    <div className="pt-6">
      <div className="flex items-center gap-4 mb-6">
        <button
          onClick={() => navigate(`/${organizationId}/settings/applications`)}
          className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        <div className="flex-1">
          <h4 className="text-2xl font-semibold">{installation.installationName || installation.appName}</h4>
          {installation.appName && installation.installationName !== installation.appName && (
            <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-1">Application: {installation.appName}</p>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-zinc-200 dark:border-zinc-800 mb-6">
        <nav className="flex gap-8">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`pb-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                activeTab === tab.id
                  ? "border-blue-500 text-blue-600 dark:text-blue-400"
                  : "border-transparent text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 hover:border-zinc-300 dark:hover:border-zinc-700"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === "overview" && (
        <div className="space-y-6">
          <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
            <div className="p-6">
              <h2 className="text-lg font-medium mb-4">Installation Details</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <h3 className="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-1">Installation ID</h3>
                  <p className="text-sm text-zinc-900 dark:text-zinc-100 font-mono">{installation.id}</p>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-1">State</h3>
                  <span
                    className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${
                      installation.state === "ready"
                        ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                        : installation.state === "error"
                          ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                          : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
                    }`}
                  >
                    {installation.state || "unknown"}
                  </span>
                  {installation.stateDescription && (
                    <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-2">{installation.stateDescription}</p>
                  )}
                </div>
              </div>
            </div>
          </div>

          {appDefinition && (
            <>
              {appDefinition.components && appDefinition.components.length > 0 && (
                <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
                  <div className="p-6">
                    <h2 className="text-lg font-medium mb-4">Components</h2>
                    <ul className="space-y-2">
                      {appDefinition.components.map((component) => (
                        <li key={component.name} className="text-sm text-zinc-700 dark:text-zinc-300">
                          • {component.label || component.name}
                        </li>
                      ))}
                    </ul>
                  </div>
                </div>
              )}

              {appDefinition.triggers && appDefinition.triggers.length > 0 && (
                <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
                  <div className="p-6">
                    <h2 className="text-lg font-medium mb-4">Triggers</h2>
                    <ul className="space-y-2">
                      {appDefinition.triggers.map((trigger) => (
                        <li key={trigger.name} className="text-sm text-zinc-700 dark:text-zinc-300">
                          • {trigger.label || trigger.name}
                        </li>
                      ))}
                    </ul>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      )}

      {/* Configuration Tab */}
      {activeTab === "configuration" && (
        <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <div className="p-6">
            {installation?.browserAction && (
              <div className="mb-6 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1">
                    {installation.browserAction.description && (
                      <p className="text-sm text-yellow-800 dark:text-yellow-200 whitespace-pre-wrap">
                        {installation.browserAction.description}
                      </p>
                    )}
                  </div>
                  <Button type="button" color="blue" onClick={handleBrowserAction} className="shrink-0">
                    <ExternalLink className="w-4 h-4 mr-2" />
                    Continue
                  </Button>
                </div>
              </div>
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
              <p className="text-sm text-zinc-500 dark:text-zinc-400">No configuration fields available.</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
