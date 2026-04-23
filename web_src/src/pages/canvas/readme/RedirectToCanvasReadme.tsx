import { Navigate, useParams, useSearchParams } from "react-router-dom";

//
// Legacy /canvases/:canvasId/readme redirect.
//
// The Canvas Readme used to be a standalone page; it now lives in a modal
// inside the canvas itself. Shared / bookmarked URLs still land here, so we
// rewrite them to the canvas with `?readme=1` — a one-shot flag consumed by
// workflowv2/index.tsx to auto-open the modal. Any pre-existing `?version=`
// is preserved so deep links into a specific canvas version still work.
//
export function RedirectToCanvasReadme() {
  const { organizationId = "", canvasId = "" } = useParams<{ organizationId: string; canvasId: string }>();
  const [searchParams] = useSearchParams();

  const nextParams = new URLSearchParams();
  nextParams.set("readme", "1");
  const versionParam = searchParams.get("version");
  if (versionParam) {
    nextParams.set("version", versionParam);
  }

  const target = `/${organizationId}/canvases/${canvasId}?${nextParams.toString()}`;
  return <Navigate to={target} replace />;
}
