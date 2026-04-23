import type { CanvasesCanvas, OrganizationsOrganization, RolesRole, SuperplaneUsersUser } from "@/api-client";
import { usePermissions } from "@/contexts/PermissionsContext";
import { useUpdateCanvasVersion } from "@/hooks/useCanvasData";
import { useCallback, useMemo } from "react";
import { buildSettingsInitialValues } from "./buildInitialValues";
import type { SettingsSavePayload } from "./types";
import { SettingsView } from "./View";

type CanvasSettingsFormProps = {
  organizationId: string;
  canvasId: string;
  versionId: string;
  canvas: CanvasesCanvas;
  organization: OrganizationsOrganization;
  organizationUsers: SuperplaneUsersUser[];
  organizationRoles: RolesRole[];
  onClose?: () => void;
};

export function CanvasSettingsForm({
  organizationId,
  canvasId,
  versionId,
  canvas,
  organization,
  organizationUsers,
  organizationRoles,
  onClose,
}: CanvasSettingsFormProps) {
  const { canAct } = usePermissions();
  const canUpdateCanvas = canAct("canvases", "update");
  const isOrgChangeManagementEnabled = organization.spec?.changeManagementEnabled ?? false;
  const updateCanvasVersionMutation = useUpdateCanvasVersion(organizationId, canvasId);

  const initialValues = useMemo(() => buildSettingsInitialValues(canvas), [canvas]);
  const availableUsers = useMemo(
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
  const availableRoles = useMemo(
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
      await updateCanvasVersionMutation.mutateAsync({
        versionId,
        name: values.name,
        description: values.description,
        nodes: canvas.spec?.nodes || [],
        edges: canvas.spec?.edges || [],
        changeManagement: values.changeManagement,
        preserveLocalCanvasState: true,
      });

      onClose?.();
    },
    [canvas.spec?.edges, canvas.spec?.nodes, onClose, updateCanvasVersionMutation, versionId],
  );

  return (
    <SettingsView
      initialValues={initialValues}
      canUpdateCanvas={canUpdateCanvas}
      orgChangeManagementEnabled={isOrgChangeManagementEnabled}
      isSaving={updateCanvasVersionMutation.isPending}
      availableUsers={availableUsers}
      availableRoles={availableRoles}
      onSave={handleSave}
    />
  );
}
