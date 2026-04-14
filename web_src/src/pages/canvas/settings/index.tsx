import { usePermissions } from "@/contexts/PermissionsContext";
import { useCanvas, useUpdateCanvas } from "@/hooks/useCanvasData";
import { useOrganization, useOrganizationRoles, useOrganizationUsers } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Loader2 } from "lucide-react";
import { useCallback, useMemo } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { buildSettingsInitialValues } from "./buildInitialValues";
import { PageHeader } from "./PageHeader";
import type { SettingsSavePayload } from "./types";
import { SettingsView } from "./View";

export function CanvasSettingsPage() {
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();
  const navigate = useNavigate();
  const { canAct } = usePermissions();
  const canReadOrg = canAct("org", "read");
  const canUpdateCanvas = canAct("canvases", "update");

  const {
    data: canvas,
    isLoading: canvasLoading,
    error: canvasError,
  } = useCanvas(organizationId || "", canvasId || "");

  const { data: organization } = useOrganization(organizationId || "", !!organizationId && canReadOrg);
  const isOrgVersioningEnabled = organization?.metadata?.versioningEnabled;

  const { data: organizationUsers = [] } = useOrganizationUsers(organizationId || "");
  const { data: organizationRoles = [] } = useOrganizationRoles(organizationId || "");

  const updateCanvasMutation = useUpdateCanvas(organizationId || "", canvasId || "");

  const isTemplate = canvas?.metadata?.isTemplate ?? false;
  const baseCanvasPath = useMemo(() => {
    if (!organizationId || !canvasId) {
      return "";
    }
    if (canvas?.metadata?.isTemplate) {
      return `/${organizationId}/templates/${canvasId}`;
    }
    return `/${organizationId}/canvases/${canvasId}`;
  }, [organizationId, canvasId, canvas?.metadata?.isTemplate]);

  usePageTitle([canvas?.metadata?.name ? `${canvas.metadata.name} · Settings` : "Canvas settings"]);

  const initialValues = useMemo(() => buildSettingsInitialValues(canvas), [canvas]);

  const approverUsers = useMemo(
    () =>
      organizationUsers
        .map((user) => {
          const id = user.metadata?.id || "";
          if (!id) {
            return null;
          }
          return {
            id,
            name: user.spec?.displayName || user.metadata?.email || id,
          };
        })
        .filter((item): item is { id: string; name: string } => !!item),
    [organizationUsers],
  );

  const approverRoles = useMemo(
    () =>
      organizationRoles
        .map((role) => {
          const name = role.metadata?.name || "";
          if (!name) {
            return null;
          }
          return {
            name,
            label: role.spec?.displayName || name,
          };
        })
        .filter((item): item is { name: string; label: string } => !!item),
    [organizationRoles],
  );

  const handleSave = useCallback(
    async (values: SettingsSavePayload) => {
      if (!canvasId) {
        return;
      }
      await updateCanvasMutation.mutateAsync({
        name: values.name,
        description: values.description,
        versioningEnabled: values.versioningEnabled,
        changeRequestApprovalConfig: values.changeRequestApprovalConfig,
      });
    },
    [canvasId, updateCanvasMutation],
  );

  if (!organizationId || !canvasId) {
    return (
      <div className="flex h-full items-center justify-center bg-slate-100 text-sm text-slate-600">
        Missing organization or canvas.
      </div>
    );
  }

  if (canvasLoading) {
    return (
      <div className="flex h-full flex-col bg-slate-100">
        <PageHeader organizationId={organizationId} />
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-slate-400" aria-label="Loading" />
        </div>
      </div>
    );
  }

  if (canvasError || !canvas) {
    return (
      <div className="flex h-full flex-col bg-slate-100">
        <PageHeader organizationId={organizationId} />
        <div className="flex flex-1 flex-col items-center justify-center gap-3 px-4 text-center">
          <p className="text-sm text-slate-600">This canvas could not be loaded.</p>
          <Link
            to={baseCanvasPath || `/${organizationId}`}
            className="text-sm font-medium text-sky-700 hover:underline"
          >
            Back to canvas
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col bg-slate-100">
      <PageHeader organizationId={organizationId} centerTitle={`${canvas.metadata?.name || "Canvas"} · Settings`} />
      <div className="min-h-0 flex-1 overflow-auto">
        <SettingsView
          initialValues={initialValues}
          canUpdateCanvas={canUpdateCanvas && !isTemplate}
          orgVersioningEnabled={isOrgVersioningEnabled}
          isSaving={updateCanvasMutation.isPending}
          availableUsers={approverUsers}
          availableRoles={approverRoles}
          onSave={handleSave}
          onBackToCanvas={() => navigate(baseCanvasPath)}
        />
      </div>
    </div>
  );
}
