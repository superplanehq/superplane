import { Eye, Rocket, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { useDeleteCanvasVersion, usePublishCanvasVersion } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

export interface DraftActionsWidgetProps {
  versionId: string;
  message?: string;
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
  onDismiss?: () => void;
}

export function DraftActionsWidget({
  versionId,
  message,
  canvasId,
  organizationId,
  isEditing,
  onDismiss,
}: DraftActionsWidgetProps) {
  const [busy, setBusy] = useState<"publish" | "discard" | null>(null);
  const publishVersion = usePublishCanvasVersion(organizationId, canvasId);
  const deleteVersion = useDeleteCanvasVersion(organizationId, canvasId);

  const handleViewInEditor = () => {
    window.dispatchEvent(new CustomEvent("agent:view-version", { detail: { versionId } }));
  };

  // Surfacing errors as toasts (instead of console.error) avoids polluting Sentry
  // with user-actionable failures coming from the publish/delete endpoints, while
  // still telling the user what went wrong.
  const runAction = async (action: "publish" | "discard", perform: () => Promise<unknown>, fallbackMessage: string) => {
    setBusy(action);
    try {
      await perform();
      onDismiss?.();
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, fallbackMessage));
    } finally {
      setBusy(null);
    }
  };

  const handlePublish = () =>
    runAction("publish", () => publishVersion.mutateAsync(versionId), "Failed to publish version.");

  const handleDiscard = () =>
    runAction("discard", () => deleteVersion.mutateAsync(versionId), "Failed to discard version.");

  return (
    <div className="flex items-center gap-2">
      {message && <span className="text-xs text-slate-600 flex-1 truncate">{message}</span>}
      {!message && <span className="text-xs text-slate-600 flex-1">Draft ready</span>}
      {!isEditing && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleViewInEditor}
          className="text-xs h-7 gap-1"
          disabled={busy !== null}
        >
          <Eye size={12} />
          See in Editor
        </Button>
      )}
      <Button
        variant="outline"
        size="sm"
        onClick={handleDiscard}
        disabled={busy !== null}
        className="text-xs h-7 gap-1 text-red-600 hover:text-red-700 hover:bg-red-50 border-red-200"
      >
        <Trash2 size={12} />
        {busy === "discard" ? "Discarding..." : "Discard"}
      </Button>
      <Button
        variant="default"
        size="sm"
        onClick={handlePublish}
        disabled={busy !== null}
        className="text-xs h-7 gap-1"
      >
        <Rocket size={12} />
        {busy === "publish" ? "Publishing..." : "Publish"}
      </Button>
    </div>
  );
}
