import { AppWindow, Loader2, X } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  useAvailableApplications,
  useInstalledApplications,
  useInstallApplication,
} from "../../../hooks/useApplications";
import { Button } from "@/ui/button";
import { ConfigurationFieldRenderer } from "../../../ui/configurationFieldRenderer";
import type { ApplicationsApplicationDefinition } from "../../../api-client/types.gen";
import { resolveIcon } from "@/lib/utils";

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
      if (result.data?.installation?.metadata?.id) {
        navigate(`/${organizationId}/settings/applications/${result.data.installation.metadata.id}`);
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
          <h4 className="text-2xl font-semibold">Applications</h4>
        </div>
      </div>

      {/* Installed Applications */}
      {installedApps.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-medium mb-4">Installed</h2>
          <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
            <table className="w-full table-fixed">
              <thead className="bg-zinc-50 dark:bg-zinc-800 border-b border-zinc-200 dark:border-zinc-700">
                <tr>
                  <th className="px-3 py-2 w-80 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                    ID
                  </th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                    Name
                  </th>
                  <th className="px-3 py-2 w-24 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                    State
                  </th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                    Application
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200 dark:divide-zinc-700">
                {[...installedApps]
                  .sort((a, b) => (a.spec?.appName || "").localeCompare(b.spec?.appName || ""))
                  .map((app) => {
                    const appDefinition = availableApps.find((a) => a.name === app.spec?.appName);
                    const appLabel = appDefinition?.label || app.spec?.appName;
                    const AppIcon = resolveIcon(appDefinition?.icon);

                    return (
                      <tr
                        key={app.metadata?.id}
                        onClick={() => navigate(`/${organizationId}/settings/applications/${app.metadata?.id}`)}
                        className="hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors cursor-pointer"
                      >
                        <td className="px-3 py-2 text-xs font-mono text-zinc-600 dark:text-zinc-400 whitespace-nowrap">
                          {app.metadata?.id}
                        </td>
                        <td className="px-3 py-2 truncate">
                          <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
                            {app.metadata?.name}
                          </div>
                        </td>
                        <td className="px-3 py-2 whitespace-nowrap">
                          <span
                            className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                              app.status?.state === "ready"
                                ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                                : app.status?.state === "error"
                                  ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                                  : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
                            }`}
                          >
                            {app.status?.state?.charAt(0).toUpperCase() + app.status?.state?.slice(1)}
                          </span>
                        </td>
                        <td className="px-3 py-2 text-sm text-zinc-600 dark:text-zinc-400 truncate">
                          <div className="flex items-center gap-2">
                            <AppIcon className="w-4 h-4" />
                            <span>{appLabel}</span>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Available Applications */}
      <div>
        <h2 className="text-lg font-medium mb-4">Available</h2>
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
              <div className="grid gap-3 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
                {availableApps.map((app) => {
                  const Icon = resolveIcon(app.icon);
                  return (
                    <div
                      key={app.name}
                      className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 hover:shadow-md transition-shadow"
                    >
                      <div className="flex items-start justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <Icon className="w-4 h-4 text-zinc-600 dark:text-zinc-400" />
                          <h3 className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
                            {app.label || app.name}
                          </h3>
                        </div>
                      </div>

                      {app.description && (
                        <p className="text-xs text-zinc-600 dark:text-zinc-400 mb-3 line-clamp-2">{app.description}</p>
                      )}

                      <Button color="blue" onClick={() => handleInstallClick(app)} className="w-full text-sm py-1.5">
                        Install
                      </Button>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Install Modal */}
      {isModalOpen &&
        selectedApplication &&
        (() => {
          const ModalIcon = resolveIcon(selectedApplication.icon);
          return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
              <div className="bg-white dark:bg-zinc-900 rounded-lg shadow-xl border border-zinc-200 dark:border-zinc-800 max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto">
                <div className="p-6">
                  <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3">
                      <ModalIcon className="w-6 h-6 text-zinc-600 dark:text-zinc-400" />
                      <h3 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
                        Install {selectedApplication.label || selectedApplication.name}
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
                      <div className="border-t border-gray-200 dark:border-zinc-700 pt-6 space-y-4">
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
                      </div>
                    )}
                  </div>

                  <div className="flex justify-end gap-3 mt-6">
                    <Button variant="outline" onClick={handleCloseModal} disabled={installMutation.isPending}>
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
          );
        })()}
    </div>
  );
}
