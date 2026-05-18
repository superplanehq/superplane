import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useSyncApp } from "@/hooks/useAppData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { CheckCircle2, ExternalLink, Loader2, RefreshCw, XCircle } from "lucide-react";
import type { AppsApp } from "@/lib/appsApi";

interface SyncIndicatorProps {
  app: AppsApp;
  canSync?: boolean;
}

export function SyncIndicator({ app, canSync = false }: SyncIndicatorProps) {
  const syncMutation = useSyncApp(app.metadata?.id ?? "");
  const syncState = app.syncState;

  if (!syncState) return null;

  const status = syncState.status ?? "synced";
  const sha = syncState.liveCommitSha ? syncState.liveCommitSha.slice(0, 7) : null;
  const remoteUrl = syncState.codeStorageRemoteUrl;

  const handleSync = async () => {
    try {
      await syncMutation.mutateAsync();
      showSuccessToast("Sync started");
    } catch {
      showErrorToast("Failed to trigger sync");
    }
  };

  const statusVariant = () => {
    switch (status.toLowerCase()) {
      case "synced":
        return "default";
      case "syncing":
        return "secondary";
      case "failed":
        return "destructive";
      default:
        return "outline";
    }
  };

  const StatusIcon = () => {
    switch (status.toLowerCase()) {
      case "synced":
        return <CheckCircle2 className="h-3 w-3" />;
      case "syncing":
        return <Loader2 className="h-3 w-3 animate-spin" />;
      case "failed":
        return <XCircle className="h-3 w-3" />;
      default:
        return null;
    }
  };

  return (
    <div className="flex items-center gap-2">
      {sha && (
        <span className="font-mono text-xs text-muted-foreground bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 rounded">
          {sha}
        </span>
      )}
      <Badge variant={statusVariant()} className="flex items-center gap-1 text-xs">
        <StatusIcon />
        {status.charAt(0).toUpperCase() + status.slice(1).toLowerCase()}
      </Badge>
      {remoteUrl && (
        <a
          href={remoteUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="text-muted-foreground hover:text-foreground"
          title="Open in Code Storage"
        >
          <ExternalLink className="h-3.5 w-3.5" />
        </a>
      )}
      {canSync && (
        <Button
          variant="ghost"
          size="sm"
          className="h-6 px-2 text-xs"
          onClick={handleSync}
          disabled={syncMutation.isPending || status.toLowerCase() === "syncing"}
          title="Sync now"
        >
          <RefreshCw className={`h-3 w-3 ${syncMutation.isPending ? "animate-spin" : ""}`} />
          Sync
        </Button>
      )}
    </div>
  );
}
