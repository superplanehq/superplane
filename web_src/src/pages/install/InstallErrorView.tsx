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
      <h2 className="mb-4 text-lg font-medium text-content-primary">Install App</h2>
      <div className="rounded-lg bg-surface-raised p-6 shadow-sm outline outline-edge-subtle">
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
