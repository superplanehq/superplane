import { Eye, Rocket } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export interface DraftActionsWidgetProps {
  versionId: string;
  message?: string;
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
}

export function DraftActionsWidget({
  versionId,
  message,
  canvasId,
  organizationId,
  isEditing,
}: DraftActionsWidgetProps) {
  const navigate = useNavigate();
  const [published, setPublished] = useState(false);
  const [publishing, setPublishing] = useState(false);

  if (published) return null;

  const handleViewInEditor = () => {
    navigate(`/${organizationId}/canvases/${canvasId}?version=${versionId}`);
  };

    const handlePublish = async () => {
    setPublishing(true);
    try {
      const response = await fetch(
        `/api/v1/canvases/${canvasId}/versions/${versionId}/publish`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
        },
      );
      if (response.ok) {
        setPublished(true);
      }
    } catch (err) {
      console.error("Failed to publish:", err);
    } finally {
      setPublishing(false);
    }
  };

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
        >
          <Eye size={12} />
          See in Editor
        </Button>
      )}
      <Button
        variant="default"
        size="sm"
        onClick={handlePublish}
        disabled={publishing}
        className={cn("text-xs h-7 gap-1 bg-violet-600 hover:bg-violet-700")}
      >
        <Rocket size={12} />
        {publishing ? "Publishing..." : "Publish"}
      </Button>
    </div>
  );
}
