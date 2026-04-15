import type { CanvasesCanvas } from "@/api-client";
import type { SettingsValues } from "./types";

export function buildSettingsInitialValues(canvas: CanvasesCanvas | undefined): SettingsValues {
  return {
    name: canvas?.metadata?.name || "",
    description: canvas?.metadata?.description || "",
    changeManagementEnabled: canvas?.spec?.changeManagement?.enabled ?? false,
    approvers: (canvas?.spec?.changeManagement?.approvals || [])
      .map((item) => {
        if (!item.type || (item.type !== "TYPE_ANYONE" && item.type !== "TYPE_USER" && item.type !== "TYPE_ROLE")) {
          return null;
        }
        return {
          type: item.type,
          userId: item.userId,
          roleName: item.roleName,
        };
      })
      .filter(
        (
          item,
        ): item is {
          type: "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE";
          userId: string | undefined;
          roleName: string | undefined;
        } => !!item,
      ),
  };
}
