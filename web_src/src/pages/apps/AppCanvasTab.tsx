import { useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useAppCanvas } from "@/hooks/useAppData";
import { Loader2 } from "lucide-react";

/**
 * Canvas tab: resolves the App's canvas ID and navigates to the canvas page.
 * The canvas page shows a "Back to App" header when opened from an App context.
 */
export function AppCanvasTab() {
  const { organizationId = "", appId = "" } = useParams<{ organizationId: string; appId: string }>();
  const navigate = useNavigate();
  const canvasQuery = useAppCanvas(appId);

  const canvasId = canvasQuery.data?.metadata?.id;

  useEffect(() => {
    if (canvasId) {
      navigate(`/${organizationId}/canvases/${canvasId}?appId=${appId}`, { replace: true });
    }
  }, [canvasId, organizationId, appId, navigate]);

  if (canvasQuery.isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-sm text-muted-foreground">Loading canvas…</span>
      </div>
    );
  }

  if (canvasQuery.error || !canvasId) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2">
        <p className="text-sm text-muted-foreground">
          {canvasQuery.error ? "Failed to load canvas." : "No canvas linked to this app yet."}
        </p>
      </div>
    );
  }

  return null;
}
