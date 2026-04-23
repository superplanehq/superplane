import { Navigate, useParams } from "react-router-dom";

export function CanvasSettingsPage() {
  const { organizationId = "", canvasId = "" } = useParams<{ organizationId: string; canvasId: string }>();

  if (!organizationId || !canvasId) {
    return <Navigate to="/" replace />;
  }

  return <Navigate to={`/${organizationId}/canvases/${canvasId}`} replace />;
}
