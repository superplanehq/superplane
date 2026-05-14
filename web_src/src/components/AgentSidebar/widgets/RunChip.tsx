import { Play } from "lucide-react";
import { useNavigate } from "react-router-dom";

interface RunChipProps {
  runId: string;
  canvasId: string;
  organizationId: string;
  label?: string;
}

export function RunChip({ runId, canvasId, organizationId, label }: RunChipProps) {
  const navigate = useNavigate();
  const shortId = runId.substring(0, 6);

  return (
    <button
      type="button"
      onClick={() => navigate(`/${organizationId}/canvases/${canvasId}?view=runs&run=${runId}`)}
      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-violet-100 text-violet-700 text-xs font-medium hover:bg-violet-200 transition-colors cursor-pointer align-middle"
      title={`Run ${runId}`}
    >
      <Play className="size-2.5 fill-current" />
      {label || `#${shortId}`}
    </button>
  );
}
