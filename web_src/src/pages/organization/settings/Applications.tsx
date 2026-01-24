import { AppWindow, Loader2, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import {
  useAvailableApplications,
  useInstalledApplications,
  useInstallApplication,
} from "../../../hooks/useApplications";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ConfigurationFieldRenderer } from "../../../ui/configurationFieldRenderer";
import type { ApplicationsApplicationDefinition } from "../../../api-client/types.gen";
import { resolveIcon, APP_LOGO_MAP, DARK_ICONS_NEEDING_INVERT } from "@/lib/utils";
import { getApiErrorMessage } from "@/utils/errors";

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
  const selectedInstructions = useMemo(() => {
    return selectedApplication?.installationInstructions?.trim();
  }, [selectedApplication?.installationInstructions]);

  const renderAppIcon = (slug: string | undefined, appName: string | undefined, className: string) => {
    const logo = appName ? APP_LOGO_MAP[appName] : undefined;
    if (logo) {
      const needsInvert = appName && DARK_ICONS_NEEDING_INVERT.includes(appName);
      return (
        <span className={className}>
          <img src={logo} alt="" className={`h-full w-full object-contain ${needsInvert ? "dark:invert" : ""}`} />
        </span>
      );
    }
    const Icon = resolveIcon(slug);
    return <Icon className={className} />;
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
    installMutation.reset();
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex justify-center items-center h-32">
          <p className="text-gray-500 dark:text-gray-400">Loading applications...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      {/* Installed Applications */}
      {installedApps.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-medium mb-4">Installed</h2>
          <div className="space-y-4">
            {[...installedApps]
              .sort((a, b) => (a.spec?.appName || "").localeCompare(b.spec?.appName || ""))
              .map((app) => {
                const appDefinition = availableApps.find((a) => a.name === app.spec?.appName);
                const appLabel = appDefinition?.label || app.spec?.appName;
                const appName = appDefinition?.name || app.spec?.appName;
                const statusLabel = app.status?.state
                  ? app.status.state.charAt(0).toUpperCase() + app.status.state.slice(1)
                  : "Unknown";

                return (
                  <div
                    key={app.metadata?.id}
                    className="bg-white dark:bg-neutral-800 border border-gray-300 dark:border-neutral-700 rounded-md p-4 flex items-start justify-between gap-4"
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-0.5 flex h-4 w-4 items-center justify-center">
                        {renderAppIcon(appDefinition?.icon, appName, "w-4 h-4 text-gray-500 dark:text-gray-400")}
                      </div>
                      <div>
                        <h3 className="text-sm font-semibold text-gray-800 dark:text-white">
                          {appLabel || app.metadata?.name || app.spec?.appName}
                        </h3>
                        {appDefinition?.description ? (
                          <p className="mt-1 text-sm text-gray-800 dark:text-gray-400">{appDefinition.description}</p>
                        ) : null}
                      </div>
                    </div>
                    <div className="flex items-start gap-6">
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                          app.status?.state === "ready"
                            ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                            : app.status?.state === "error"
                              ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                              : "bg-orange-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
                        }`}
                      >
                        {statusLabel}
                      </span>
                      <Button
                        variant="outline"
                        onClick={() =>
                          navigate(`/${organizationId}/settings/applications/${app.metadata?.id}`, {
                            state: { tab: "configuration" },
                          })
                        }
                        className="text-sm py-1.5 self-start"
                      >
                        Configure...
                      </Button>
                    </div>
                  </div>
                );
              })}
          </div>
        </div>
      )}

      {/* Available Applications */}
      <div>
        <h2 className="text-lg font-medium mb-4">Available</h2>
        <div>
          {availableApps.filter((app) => !installedApps.some((installed) => installed.spec?.appName === app.name))
            .length === 0 ? (
            <div className="text-center py-12">
              <AppWindow className="w-6 h-6 text-gray-800 mx-auto mb-2" />
              <p className="text-sm text-gray-800">You&apos;ve installed all applications!</p>
            </div>
          ) : (
            <div className="space-y-4">
              {[...availableApps]
                .filter((app) => !installedApps.some((installed) => installed.spec?.appName === app.name))
                .sort((a, b) => (a.label || a.name || "").localeCompare(b.label || b.name || ""))
                .map((app) => {
                  const appName = app.name;
                  return (
                    <div
                      key={app.name}
                      className="bg-white dark:bg-neutral-800 border border-gray-300 dark:border-neutral-700 rounded-md p-4 flex items-start justify-between gap-4"
                    >
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 flex h-4 w-4 items-center justify-center">
                          {renderAppIcon(app.icon, appName, "w-4 h-4 text-gray-500 dark:text-gray-400")}
                        </div>
                        <div>
                          <h3 className="text-sm font-semibold text-gray-800 dark:text-white">
                            {app.label || app.name}
                          </h3>
                          {app.description ? (
                            <p className="mt-1 text-sm text-gray-800 dark:text-gray-400">{app.description}</p>
                          ) : null}
                        </div>
                      </div>

                      <Button
                        color="blue"
                        onClick={() => handleInstallClick(app)}
                        className="text-sm py-1.5 self-start"
                      >
                        Install
                      </Button>
                    </div>
                  );
                })}
            </div>
          )}
        </div>
      </div>

      {/* Install Modal */}
      <Dialog open={isModalOpen && !!selectedApplication} onOpenChange={(open) => !open && handleCloseModal()}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto" showCloseButton={false}>
          {selectedApplication && (
            <>
              <DialogHeader>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    {renderAppIcon(selectedApplication.icon, selectedApplication.name, "w-6 h-6 text-muted-foreground")}
                    <DialogTitle>Install {selectedApplication.label || selectedApplication.name}</DialogTitle>
                  </div>
                  <button
                    onClick={handleCloseModal}
                    className="text-muted-foreground hover:text-foreground"
                    disabled={installMutation.isPending}
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              </DialogHeader>

              <div className="space-y-4">
                {selectedInstructions && (
                  <div className="rounded-md border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900 dark:border-blue-900/40 dark:bg-blue-950/40 dark:text-blue-100 [&_ol]:list-decimal [&_ol]:ml-5 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-5 [&_ul]:space-y-1">
                    <ReactMarkdown>{selectedInstructions}</ReactMarkdown>
                  </div>
                )}
                {/* Installation Name Field */}
                <div>
                  <Label className="mb-2">
                    Installation Name
                    <span className="text-muted-foreground ml-1">*</span>
                  </Label>
                  <p className="text-xs text-muted-foreground mb-2">A unique name for this installation</p>
                  <Input
                    type="text"
                    value={installationName}
                    onChange={(e) => setInstallationName(e.target.value)}
                    placeholder="e.g., my-app-integration"
                    required
                  />
                </div>

                {/* Configuration Fields */}
                {selectedApplication.configuration && selectedApplication.configuration.length > 0 && (
                  <div className="border-t border-border pt-6 space-y-4">
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
                          organizationId={organizationId}
                        />
                      );
                    })}
                  </div>
                )}
              </div>

              <div className="flex justify-start gap-3 mt-6">
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
                <Button variant="outline" onClick={handleCloseModal} disabled={installMutation.isPending}>
                  Cancel
                </Button>
              </div>

              {installMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to install application: {getApiErrorMessage(installMutation.error)}
                  </p>
                </div>
              )}
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
