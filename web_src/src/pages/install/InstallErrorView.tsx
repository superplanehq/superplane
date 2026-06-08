import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { getUsageLimitNotice } from "@/lib/usageLimits";
import { InstallPageHeader } from "./InstallPageHeader";
import { InstallShell } from "./InstallShell";

interface InstallErrorViewProps {
  loadError: string | null;
}

export function InstallErrorView({ loadError }: InstallErrorViewProps) {
  const usageLimitNotice = loadError ? getUsageLimitNotice(loadError) : null;

  return (
    <InstallShell>
      <InstallPageHeader title="Install App" description="Add a pre-built app from GitHub to your organization." />
      <div className="rounded-lg bg-white p-6 shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-800">
        {usageLimitNotice ? (
          <UsageLimitAlert notice={usageLimitNotice} />
        ) : (
          <Alert variant="destructive">
            <AlertTitle>Unable to install app</AlertTitle>
            <AlertDescription>{loadError || "Unable to load app installation details."}</AlertDescription>
          </Alert>
        )}
      </div>
    </InstallShell>
  );
}
