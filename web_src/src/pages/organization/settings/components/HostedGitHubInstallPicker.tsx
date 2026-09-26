import { Button } from "@/components/ui/button";
import {
  bindHostedGitHubInstallation,
  hostedGitHubInstallURL,
  type PendingGitHubInstallation,
} from "@/lib/hostedGitHubInstall";
import { useState } from "react";

interface HostedGitHubInstallPickerProps {
  installations: PendingGitHubInstallation[];
  state: string;
  appSlug: string;
}

export function HostedGitHubInstallPicker({ installations, state, appSlug }: HostedGitHubInstallPickerProps) {
  const [bindingRepositoryId, setBindingRepositoryId] = useState<string>();
  const [error, setError] = useState("");

  const connectRepository = async (installationId: string, repositoryId: string) => {
    setBindingRepositoryId(repositoryId);
    setError("");
    try {
      await bindHostedGitHubInstallation(state, installationId, repositoryId);
      window.location.reload();
    } catch {
      setError("SuperPlane could not connect this repository. Try again.");
      setBindingRepositoryId(undefined);
    }
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
      <div className="p-6 space-y-4">
        <div>
          <h2 className="text-lg font-medium">Select a GitHub repository</h2>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
            Select a repository where you have write access. SuperPlane will use only the repository you select.
          </p>
        </div>
        <div className="space-y-4">
          {installations.map((installation) => (
            <div key={installation.id} className="space-y-2">
              <p className="text-sm font-medium">{installation.accountLogin}</p>
              {installation.repositories.map((repository) => (
                <Button
                  key={repository.id}
                  type="button"
                  className="w-full justify-start"
                  disabled={bindingRepositoryId !== undefined}
                  onClick={() => void connectRepository(installation.id, repository.id)}
                >
                  {bindingRepositoryId === repository.id ? "Connecting…" : `Use ${repository.name}`}
                </Button>
              ))}
            </div>
          ))}
        </div>
        {error ? (
          <p className="text-sm text-red-600 dark:text-red-400" role="alert">
            {error}
          </p>
        ) : null}
        {appSlug !== "" && (
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Do not see your GitHub account or organization?{" "}
            <a
              href={hostedGitHubInstallURL(appSlug, state)}
              className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
            >
              Install the GitHub App there.
            </a>
          </p>
        )}
      </div>
    </div>
  );
}
