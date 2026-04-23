import type { CanvasesCanvas, OrganizationsOrganization } from "@/api-client";
import { usePermissions } from "@/contexts/PermissionsContext";
import { useCanvas, useCanvasReadme, useCreateCanvasChangeRequest, useUpdateCanvasReadme } from "@/hooks/useCanvasData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Loader2 } from "lucide-react";
import { useParams, useSearchParams } from "react-router-dom";
import { PageHeader } from "../settings/PageHeader";
import { ReadmeView } from "./ReadmeView";

export function CanvasReadmePage() {
  const { organizationId = "", canvasId = "" } = useParams<{ organizationId: string; canvasId: string }>();

  const { canAct } = usePermissions();
  const canReadOrg = !!organizationId && canAct("org", "read");

  const { data: organization } = useOrganization(organizationId, canReadOrg);
  const { data: canvas, isLoading: canvasLoading, error: canvasError } = useCanvas(organizationId, canvasId);

  if (!organizationId || !canvasId) {
    return <ErrorView organizationId={organizationId} error="Missing organization or canvas." />;
  }

  if (canvasLoading || !organization) {
    return <LoadingView organizationId={organizationId} />;
  }

  if (canvasError || !canvas) {
    return <ErrorView organizationId={organizationId} error="This canvas could not be loaded." />;
  }

  return <NormalView canvas={canvas} organization={organization} />;
}

function LoadingView({ organizationId }: { organizationId: string }) {
  usePageTitle(["Canvas readme"]);

  return (
    <div className="flex h-full flex-col bg-slate-100">
      <PageHeader organizationId={organizationId} title="" />
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-slate-400" aria-label="Loading" />
      </div>
    </div>
  );
}

function ErrorView({ organizationId, error }: { organizationId?: string; error: string }) {
  usePageTitle(["Canvas readme"]);

  return (
    <div className="flex h-full flex-col bg-slate-100">
      {organizationId && <PageHeader organizationId={organizationId} title="" />}
      <div className="flex flex-1 items-center justify-center">
        <p className="text-sm text-slate-600">{error}</p>
      </div>
    </div>
  );
}

function NormalView({ canvas, organization }: { canvas: CanvasesCanvas; organization: OrganizationsOrganization }) {
  const orgId = organization.metadata!.id!;
  const resolvedCanvasId = canvas.metadata!.id!;
  const canvasName = canvas.metadata?.name || "Canvas";

  usePageTitle([`${canvasName} · Readme`]);

  const { canAct } = usePermissions();
  const canUpdateCanvas = canAct("canvases", "update");

  const changeManagementEnabled = canvas.spec?.changeManagement?.enabled ?? false;

  const [searchParams] = useSearchParams();
  const versionParam = searchParams.get("version") ?? "";

  const liveReadme = useCanvasReadme(resolvedCanvasId, "");
  const draftReadme = useCanvasReadme(resolvedCanvasId, "draft", canUpdateCanvas);

  const updateReadmeMutation = useUpdateCanvasReadme(resolvedCanvasId);
  const createChangeRequestMutation = useCreateCanvasChangeRequest(orgId, resolvedCanvasId);

  //
  // The readme page mirrors the canvas's Live / Editor split. When the Readme
  // button is clicked from the Editor tab, the canvas forwards its current
  // ?version=<draft-id>. We enter edit mode iff that id matches the caller's
  // own draft (and they have update permission). Anything else (no param,
  // published version id, a draft owned by someone else, missing permission)
  // collapses to the read-only live view.
  //
  const mode: "live" | "edit" =
    canUpdateCanvas &&
    !!versionParam &&
    draftReadme.data?.versionId === versionParam
      ? "edit"
      : "live";

  const nodes = canvas.spec?.nodes ?? [];
  const nodesBySlug: Record<string, string> = {};
  const nodeIdBySlug: Record<string, string> = {};
  for (const node of nodes) {
    const slug = node.name || node.id;
    if (!slug) continue;
    nodesBySlug[slug] = node.name || slug;
    if (node.id) {
      nodeIdBySlug[slug] = node.id;
    }
  }

  const linkForNode = (slug: string) => {
    const id = nodeIdBySlug[slug] ?? slug;
    const suffix = mode === "edit" && versionParam ? `&version=${encodeURIComponent(versionParam)}` : "";
    return `/${orgId}/canvases/${resolvedCanvasId}?node=${encodeURIComponent(id)}${suffix}`;
  };

  const backToCanvasHref =
    mode === "edit" && versionParam
      ? `/${orgId}/canvases/${resolvedCanvasId}?version=${encodeURIComponent(versionParam)}`
      : `/${orgId}/canvases/${resolvedCanvasId}`;

  return (
    <div className="flex h-full min-h-0 flex-col bg-slate-100">
      <PageHeader organizationId={orgId} title={`${canvasName} · Readme`} />

      <div className="min-h-0 flex-1 overflow-auto">
        <ReadmeView
          mode={mode}
          backToCanvasHref={backToCanvasHref}
          canvasName={canvasName}
          changeManagementEnabled={changeManagementEnabled}
          liveContent={liveReadme.data?.content ?? ""}
          draftContent={draftReadme.data?.content ?? ""}
          isLoadingLive={liveReadme.isLoading}
          isLoadingDraft={canUpdateCanvas && draftReadme.isLoading}
          isSavingDraft={updateReadmeMutation.isPending}
          isCreatingChangeRequest={createChangeRequestMutation.isPending}
          draftVersionId={draftReadme.data?.versionId}
          nodes={nodesBySlug}
          linkFor={linkForNode}
          onSaveDraft={async (content) => {
            await updateReadmeMutation.mutateAsync({ content });
          }}
          onCreateChangeRequest={async ({ title, description }) => {
            const versionId = draftReadme.data?.versionId;
            if (!versionId) {
              return;
            }
            await createChangeRequestMutation.mutateAsync({ versionId, title, description });
          }}
        />
      </div>
    </div>
  );
}
