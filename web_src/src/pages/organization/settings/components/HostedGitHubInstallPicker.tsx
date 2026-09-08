import { Button } from "@/components/ui/button";
import {
  hostedGitHubBindPath,
  hostedGitHubInstallURL,
  type PendingGitHubInstallation,
} from "@/lib/hostedGitHubInstall";

interface HostedGitHubInstallPickerProps {
  installations: PendingGitHubInstallation[];
  state: string;
  appSlug: string;
}

export function HostedGitHubInstallPicker({ installations, state, appSlug }: HostedGitHubInstallPickerProps) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
      <div className="p-6 space-y-4">
        <div>
          <h2 className="text-lg font-medium">Select a GitHub account</h2>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
            The SuperPlane GitHub App is already installed on these accounts.
          </p>
        </div>
        <div className="space-y-2">
          {installations.map((installation) => (
            <Button
              key={installation.id}
              type="button"
              className="w-full justify-start"
              onClick={() => {
                window.location.assign(hostedGitHubBindPath(state, installation.id));
              }}
            >
              Use {installation.accountLogin}
            </Button>
          ))}
        </div>
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
