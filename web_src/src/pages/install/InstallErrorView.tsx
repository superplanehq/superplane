import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { getUsageLimitNotice } from "@/lib/usageLimits";
import { InstallShell } from "./InstallShell";

interface InstallErrorViewProps {
  loadError: string | null;
}

export function InstallErrorView({ loadError }: InstallErrorViewProps) {
  const usageLimitNotice = loadError ? getUsageLimitNotice(loadError) : null;

  return (
    <InstallShell>
      <h2 className="mb-4 text-lg font-medium text-slate-900">Install App</h2>
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
