import { AppWindow, ArrowLeft, ExternalLink, Loader2 } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { useApplicationInstallation } from "../../../hooks/useApplications";
import { Button } from "@/ui/button";

interface ApplicationDetailsProps {
  organizationId: string;
}

export function ApplicationDetails({ organizationId }: ApplicationDetailsProps) {
  const navigate = useNavigate();
  const { installationId } = useParams<{ installationId: string }>();

  const { data: installation, isLoading, error } = useApplicationInstallation(
    organizationId,
    installationId || ""
  );

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
          <h1 className="text-2xl font-semibold">Application Details</h1>
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
          <h1 className="text-2xl font-semibold">Application Details</h1>
        </div>
        <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
          <p className="text-zinc-600 dark:text-zinc-400">Application installation not found</p>
        </div>
      </div>
    );
  }

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
          <h1 className="text-2xl font-semibold">{installation.installationName || installation.appLabel || installation.appName}</h1>
          {installation.appName && installation.installationName !== installation.appName && (
            <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-1">Application: {installation.appLabel || installation.appName}</p>
          )}
        </div>
      </div>

      {/* Installation Info */}
      <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 mb-6">
        <div className="p-6">
          <div className="flex items-start gap-4">
            <AppWindow className="w-12 h-12 text-zinc-600 dark:text-zinc-400" />
            <div className="flex-1">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <h3 className="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-1">Status</h3>
                  <span
                    className={`inline-flex px-3 py-1 text-sm font-medium rounded-full ${
                      installation.state === "ready"
                        ? "bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200"
                        : installation.state === "error"
                        ? "bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200"
                        : installation.state === "in-progress"
                        ? "bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200"
                        : "bg-zinc-100 dark:bg-zinc-800 text-zinc-800 dark:text-zinc-200"
                    }`}
                  >
                    {installation.state || "unknown"}
                  </span>
                  {installation.stateDescription && (
                    <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-2">{installation.stateDescription}</p>
                  )}
                </div>
                <div>
                  <h3 className="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-1">Installation ID</h3>
                  <p className="text-sm text-zinc-900 dark:text-zinc-100 font-mono">{installation.id}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Browser Action */}
      {installation.browserAction && (
        <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800 mb-6">
          <div className="p-6">
            <h2 className="text-lg font-medium mb-4">Complete Setup</h2>
            <p className="text-sm text-zinc-600 dark:text-zinc-400 mb-4">
              Complete the {installation.appLabel || installation.appName} setup by submitting the registration form.
            </p>
            <Button
              color="blue"
              onClick={handleBrowserAction}
              className="flex items-center gap-2"
            >
              <ExternalLink className="w-4 h-4" />
              Register {installation.appLabel || installation.appName} App
            </Button>
            <div className="mt-4 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-md border border-blue-200 dark:border-blue-800">
              <p className="text-sm text-blue-800 dark:text-blue-200">
                <strong>Note:</strong> Clicking the button above will open a new window and submit your app registration
                to {installation.appLabel || installation.appName}. After completing the setup, you'll be redirected back to continue the installation.
              </p>
            </div>
            {installation.browserAction.method && (
              <div className="mt-4 p-4 bg-zinc-50 dark:bg-zinc-800 rounded-md">
                <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-2">Details:</p>
                <div className="space-y-1 text-xs">
                  <p className="text-zinc-700 dark:text-zinc-300">
                    <span className="font-medium">URL:</span> {installation.browserAction.url}
                  </p>
                  <p className="text-zinc-700 dark:text-zinc-300">
                    <span className="font-medium">Method:</span> {installation.browserAction.method}
                  </p>
                  {installation.browserAction.formFields &&
                   Object.keys(installation.browserAction.formFields).length > 0 && (
                    <div>
                      <p className="font-medium text-zinc-700 dark:text-zinc-300 mt-2">Form Fields:</p>
                      <ul className="list-disc list-inside ml-2">
                        {Object.entries(installation.browserAction.formFields).map(([key, value]) => (
                          <li key={key} className="text-zinc-700 dark:text-zinc-300">
                            {key}: {String(value)}
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Configuration */}
      {installation.configuration && Object.keys(installation.configuration).length > 0 && (
        <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <div className="p-6">
            <h2 className="text-lg font-medium mb-4">Configuration</h2>
            <div className="space-y-2">
              {Object.entries(installation.configuration).map(([key, value]) => (
                <div key={key} className="flex justify-between py-2 border-b border-zinc-100 dark:border-zinc-800 last:border-0">
                  <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">{key}</span>
                  <span className="text-sm text-zinc-600 dark:text-zinc-400">
                    {typeof value === "string" && key.toLowerCase().includes("secret")
                      ? "••••••••"
                      : String(value)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
