import type { CanvasesCanvas, OrganizationsOrganization, RolesRole, SuperplaneUsersUser } from "@/api-client";
import { usePermissions } from "@/contexts/PermissionsContext";
import { useCanvas, useUpdateCanvas } from "@/hooks/useCanvasData";
import { useOrganization, useOrganizationRoles, useOrganizationUsers } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Loader2 } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { buildSettingsInitialValues } from "./buildInitialValues";
import { PageHeader } from "./PageHeader";
import type { SettingsSavePayload } from "./types";
import { SettingsView } from "./View";

export function CanvasSettingsPage() {
  const { organizationId = "", canvasId = "" } = useParams<{ organizationId: string; canvasId: string }>();

  const { canAct } = usePermissions();
  const canReadOrg = !!organizationId && canAct("org", "read");

  const { data: organization } = useOrganization(organizationId, canReadOrg);
  const { data: canvas, isLoading: canvasLoading, error: canvasError } = useCanvas(organizationId, canvasId);

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
  const baseCanvasPath = `/${orgId}/canvases/${resolvedCanvasId}`;
  const isOrgChangeManagementEnabled = organization?.metadata?.changeManagementEnabled ?? false;

  usePageTitle([`${canvasName} · Settings`]);

  const { canAct } = usePermissions();
  const canUpdateCanvas = canAct("canvases", "update");

  const { data: organizationUsers = [] } = useOrganizationUsers(orgId);
  const { data: organizationRoles = [] } = useOrganizationRoles(orgId);

  const updateCanvasMutation = useUpdateCanvas(orgId, resolvedCanvasId);

  const initialValues = useMemo(() => buildSettingsInitialValues(canvas), [canvas]);
  const approverUsers = useApproverUsers(organizationUsers);
  const approverRoles = useApproverRoles(organizationRoles);
  const onSave = useSaveCallback(resolvedCanvasId, orgId);

  return (
    <div className="flex h-full min-h-0 flex-col bg-slate-100">
      <PageHeader organizationId={orgId} title={`${canvasName} · Settings`} />

      <div className="min-h-0 flex-1 overflow-auto">
        <SettingsView
          initialValues={initialValues}
          canUpdateCanvas={canUpdateCanvas}
          orgChangeManagementEnabled={isOrgChangeManagementEnabled}
          isSaving={updateCanvasMutation.isPending}
          availableUsers={approverUsers}
          availableRoles={approverRoles}
          onSave={onSave}
          onBackToCanvas={() => navigate(baseCanvasPath)}
        />
      </div>
    </div>
  );
}

function useApproverUsers(organizationUsers: SuperplaneUsersUser[]) {
  return useMemo(
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
}

function useApproverRoles(organizationRoles: RolesRole[]) {
  return useMemo(
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
}

function useSaveCallback(canvasId: string, organizationId: string): (values: SettingsSavePayload) => Promise<void> {
  const updateCanvasMutation = useUpdateCanvas(organizationId, canvasId);

  return useCallback(
    async (values: SettingsSavePayload) => {
      if (!canvasId) {
        return;
      }
      await updateCanvasMutation.mutateAsync({
        name: values.name,
        description: values.description,
        changeManagementEnabled: values.changeManagementEnabled,
        changeRequestApprovalConfig: values.changeRequestApprovalConfig,
      });
    },
    [canvasId, updateCanvasMutation],
  );
}
