import type { CanvasesCanvas, OrganizationsOrganization } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useCanvas, useUpdateCanvas } from "@/hooks/useCanvasData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { Loader2 } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { appPath } from "@/lib/appPaths";
import { buildSettingsInitialValues } from "./buildInitialValues";
import { PageHeader } from "./PageHeader";
import type { SettingsSavePayload } from "./types";
import { SettingsView } from "./View";

export function CanvasSettingsPage() {
  const { organizationId = "", appId = "" } = useParams<{ organizationId: string; appId: string }>();
  const canvasId = appId;

  const { canAct } = usePermissions();
  const canReadOrg = !!organizationId && canAct("org", "read");

  const { data: organization } = useOrganization(organizationId, canReadOrg);
  const { data: canvas, isLoading: canvasLoading, error: canvasError } = useCanvas(organizationId, canvasId);

  useReportPageReady(!canvasLoading && !!canvas && !!organizationId && !!organization, {
    failed: !!canvasError,
  });

  if (!organizationId || !canvasId || !organization) {
    return <ErrorView organizationId={organizationId} error="Missing organization or canvas." />;
  }

  if (canvasLoading) {
    return <LoadingView organizationId={organizationId} />;
  }

  if (canvasError || !canvas) {
    return <ErrorView organizationId={organizationId} error="This canvas could not be loaded." />;
  }

  return <NormalView canvas={canvas} organization={organization} />;
}

function LoadingView({ organizationId }: { organizationId: string }) {
  usePageTitle(["Canvas settings"]);

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
  usePageTitle(["Canvas settings"]);

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
  const navigate = useNavigate();

  const orgId = organization.metadata!.id!;
  const resolvedCanvasId = canvas.metadata!.id!;
  const canvasName = canvas.metadata?.name || "Canvas";
  const baseCanvasPath = appPath(orgId, resolvedCanvasId);

  usePageTitle([`${canvasName} · Settings`]);

  const { canAct } = usePermissions();
  const canUpdateCanvas = canAct("canvases", "update");

  const updateCanvasMutation = useUpdateCanvas(orgId, resolvedCanvasId);

  const initialValues = useMemo(() => buildSettingsInitialValues(canvas), [canvas]);
  const onSave = useSaveCallback(resolvedCanvasId, orgId);

  return (
    <div className="flex h-full min-h-0 flex-col bg-slate-100">
      <PageHeader organizationId={orgId} title={`${canvasName} · Settings`} />

      <div className="min-h-0 flex-1 overflow-auto">
        <SettingsView
          initialValues={initialValues}
          canUpdateCanvas={canUpdateCanvas}
          isSaving={updateCanvasMutation.isPending}
          onSave={onSave}
          onBackToCanvas={() => navigate(baseCanvasPath)}
        />
      </div>
    </div>
  );
}

function useSaveCallback(canvasId: string, organizationId: string): (values: SettingsSavePayload) => Promise<void> {
  const navigate = useNavigate();
  const updateCanvasMutation = useUpdateCanvas(organizationId, canvasId);
  const baseCanvasPath = appPath(organizationId, canvasId);

  return useCallback(
    async (values: SettingsSavePayload) => {
      if (!canvasId) {
        return;
      }
      await updateCanvasMutation.mutateAsync({
        name: values.name,
        description: values.description,
      });

      navigate(baseCanvasPath, { replace: true });
    },
    [canvasId, updateCanvasMutation, navigate, baseCanvasPath],
  );
}
