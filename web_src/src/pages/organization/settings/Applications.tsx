import { AppWindow, Puzzle, Zap, Settings, Check, Loader2, X, Edit } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAvailableApplications, useInstalledApplications, useInstallApplication } from "../../../hooks/useApplications";
import { Button } from "../../../components/Button/button";
import { ConfigurationFieldRenderer } from "../../../ui/configurationFieldRenderer";
import type { ApplicationsApplicationDefinition } from "../../../api-client/types.gen";

interface ApplicationsProps {
  organizationId: string;
}

export function Applications({ organizationId }: ApplicationsProps) {
  const navigate = useNavigate();
  const [selectedApplication, setSelectedApplication] = useState<ApplicationsApplicationDefinition | null>(null);
  const [installationName, setInstallationName] = useState("");
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [isModalOpen, setIsModalOpen] = useState(false);

  const { data: availableApps = [], isLoading: loadingAvailable } = useAvailableApplications();
  const { data: installedApps = [], isLoading: loadingInstalled } = useInstalledApplications(organizationId);
  const installMutation = useInstallApplication(organizationId);

  const isLoading = loadingAvailable || loadingInstalled;

  // Check if an application is already installed
  const isInstalled = (appName: string) => {
    return installedApps.some((installed) => installed.appName === appName);
  };

  const handleInstallClick = (app: ApplicationsApplicationDefinition) => {
    setSelectedApplication(app);
    setInstallationName(app.name || "");
    setConfiguration({});
    setIsModalOpen(true);
  };

  const handleConfigurationChange = (fieldName: string, value: unknown) => {
    setConfiguration((prev) => ({
      ...prev,
      [fieldName]: value,
    }));
  };

  const handleInstall = async () => {
    if (!selectedApplication?.name) return;

    try {
      const result = await installMutation.mutateAsync({
        appName: selectedApplication.name,
        installationName: installationName,
        configuration,
      });
      setIsModalOpen(false);
      setSelectedApplication(null);
      setInstallationName("");
      setConfiguration({});

      // Redirect to the application installation details page
      if (result.data?.installation?.id) {
        navigate(`/${organizationId}/settings/applications/${result.data.installation.id}`);
      }
    } catch (error) {
      console.error("Failed to install application:", error);
    }
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setSelectedApplication(null);
    setInstallationName("");
    setConfiguration({});
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <h1 className="text-2xl font-semibold mb-6">Applications</h1>
        <div className="flex justify-center items-center h-32">
          <p className="text-zinc-500 dark:text-zinc-400">Loading applications...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      <div className="flex justify-between items-start mb-6">
        <div>
          <h1 className="text-2xl font-semibold">Applications</h1>
          <p className="text-zinc-600 dark:text-zinc-400 mt-2">
            Manage installed applications and discover new ones for your organization
          </p>
        </div>
      </div>

      {/* Installed Applications */}
      {installedApps.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-medium mb-4">Installed Applications</h2>
          <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
            <div className="p-6">
              <div className="space-y-4">
                {installedApps.map((app) => (
                  <div
                    key={app.id}
                    className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <AppWindow className="w-5 h-5 text-zinc-600 dark:text-zinc-400" />
                        <div>
                          <h3 className="font-medium text-zinc-900 dark:text-zinc-100">
                            {app.installationName || app.appName}
                          </h3>
                          {app.appName && app.installationName !== app.appName && (
                            <p className="text-xs text-zinc-500 dark:text-zinc-400">App: {app.appName}</p>
                          )}
                          {app.state && (
                            <p className="text-sm text-zinc-600 dark:text-zinc-400">Status: {app.state}</p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="px-3 py-1 text-xs font-medium bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 rounded-full flex items-center gap-1">
                          <Check className="w-3 h-3" />
                          Installed
                        </span>
                        <button
                          onClick={() => navigate(`/${organizationId}/settings/applications/${app.id}`)}
                          className="p-2 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded transition-colors"
                          title="Edit application"
                        >
                          <Edit className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Available Applications */}
      <div>
        <h2 className="text-lg font-medium mb-4">Available Applications</h2>
        <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <div className="p-6">
            {availableApps.length === 0 ? (
              <div className="text-center py-12">
                <AppWindow className="w-12 h-12 text-zinc-400 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-2">No applications available</h3>
                <p className="text-zinc-600 dark:text-zinc-400">
                  There are currently no applications available to install
                </p>
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {availableApps.map((app) => {
                  const installed = isInstalled(app.name || "");
                  return (
                    <div
                      key={app.name}
                      className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4 hover:shadow-md transition-shadow"
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3">
                          <AppWindow className="w-5 h-5 text-zinc-600 dark:text-zinc-400" />
                          <h3 className="font-medium text-zinc-900 dark:text-zinc-100">{app.name}</h3>
                        </div>
                      </div>

                      <div className="space-y-2 text-sm text-zinc-600 dark:text-zinc-400 mb-4">
                        {app.components && app.components.length > 0 && (
                          <div className="flex items-center gap-2">
                            <Puzzle className="w-4 h-4" />
                            <span>{app.components.length} component{app.components.length !== 1 ? "s" : ""}</span>
                          </div>
                        )}
                        {app.triggers && app.triggers.length > 0 && (
                          <div className="flex items-center gap-2">
                            <Zap className="w-4 h-4" />
                            <span>{app.triggers.length} trigger{app.triggers.length !== 1 ? "s" : ""}</span>
                          </div>
                        )}
                        {app.configuration && app.configuration.length > 0 && (
                          <div className="flex items-center gap-2">
                            <Settings className="w-4 h-4" />
                            <span>{app.configuration.length} configuration field{app.configuration.length !== 1 ? "s" : ""}</span>
                          </div>
                        )}
                      </div>

                      {installed ? (
                        <div className="flex items-center gap-2 text-sm text-green-700 dark:text-green-400">
                          <Check className="w-4 h-4" />
                          Installed
                        </div>
                      ) : (
                        <Button
                          color="blue"
                          onClick={() => handleInstallClick(app)}
                          className="w-full"
                        >
                          Install
                        </Button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Install Modal */}
      {isModalOpen && selectedApplication && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-zinc-900 rounded-lg shadow-xl border border-zinc-200 dark:border-zinc-800 max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                  <AppWindow className="w-6 h-6 text-zinc-600 dark:text-zinc-400" />
                  <h3 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
                    Install {selectedApplication.name}
                  </h3>
                </div>
                <button
                  onClick={handleCloseModal}
                  className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                  disabled={installMutation.isPending}
                >
                  <X className="w-6 h-6" />
                </button>
              </div>

              <div className="space-y-4">
                {/* Installation Name Field */}
                <div>
                  <label className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                    Installation Name
                    <span className="text-red-500 ml-1">*</span>
                  </label>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400 mb-2">
                    A unique name for this installation
                  </p>
                  <input
                    type="text"
                    value={installationName}
                    onChange={(e) => setInstallationName(e.target.value)}
                    className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="e.g., my-app-integration"
                    required
                  />
                </div>

                {/* Configuration Fields */}
                {selectedApplication.configuration && selectedApplication.configuration.length > 0 && (
                  <>
                    <div className="border-t border-zinc-200 dark:border-zinc-700 pt-4">
                      <h4 className="text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-4">Configuration</h4>
                    </div>
                    {selectedApplication.configuration.map((field) => {
                      if (!field.name) return null;
                      return (
                        <ConfigurationFieldRenderer
                          key={field.name}
                          field={field}
                          value={configuration[field.name]}
                          onChange={(value) => handleConfigurationChange(field.name || "", value)}
                          allValues={configuration}
                          domainId={organizationId}
                          domainType="DOMAIN_TYPE_ORGANIZATION"
                        />
                      );
                    })}
                  </>
                )}
              </div>

              <div className="flex justify-end gap-3 mt-6">
                <Button
                  color="zinc"
                  onClick={handleCloseModal}
                  disabled={installMutation.isPending}
                >
                  Cancel
                </Button>
                <Button
                  color="blue"
                  onClick={handleInstall}
                  disabled={installMutation.isPending || !installationName.trim()}
                  className="flex items-center gap-2"
                >
                  {installMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Installing...
                    </>
                  ) : (
                    "Install"
                  )}
                </Button>
              </div>

              {installMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to install application. Please try again.
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
