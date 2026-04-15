import type { ApproverFieldErrors, ApproverValidationResult, SettingsApprover } from "./types";

export const EMPTY_SELECT_VALUE = "__empty__";

export function normalizeApprovers(items?: SettingsApprover[]): SettingsApprover[] {
  const normalized = (items || []).map((item) => ({
    type: item.type,
    userId: item.userId,
    roleName: item.roleName,
  }));
  if (normalized.length > 0) {
    return normalized;
  }

  return [{ type: "TYPE_USER", userId: "" }];
}

export function validateApproverConfig(
  approvers: SettingsApprover[],
  availableUsers: Array<{ id: string; name: string }>,
  availableRoles: Array<{ name: string; label: string }>,
): ApproverValidationResult {
  if (approvers.length === 0) {
    return {
      formErrors: ["at least one approver is required"],
      itemErrors: [],
    };
  }

  const formErrors: string[] = [];
  const itemErrors: ApproverFieldErrors[] = approvers.map(() => ({}));
  const availableUserIDs = new Set(availableUsers.map((user) => user.id));
  const availableRoleNames = new Set(availableRoles.map((role) => role.name));
  let hasAnyUserApprover = false;
  const seenUsers = new Set<string>();
  const seenRoles = new Set<string>();

  approvers.forEach((approver, index) => {
    switch (approver.type) {
      case "TYPE_ANYONE": {
        if (hasAnyUserApprover) {
          itemErrors[index].type = "Duplicate any-user approver is not allowed";
        } else {
          hasAnyUserApprover = true;
        }
        break;
      }

      case "TYPE_USER": {
        const userId = (approver.userId || "").trim();
        if (!userId) {
          itemErrors[index].userId = "User is required";
        } else if (!availableUserIDs.has(userId)) {
          itemErrors[index].userId = "Selected user was not found in this organization";
        } else if (seenUsers.has(userId)) {
          itemErrors[index].userId = "Duplicate user approver is not allowed";
        } else {
          seenUsers.add(userId);
        }
        break;
      }

      case "TYPE_ROLE": {
        const roleName = (approver.roleName || "").trim();
        if (!roleName) {
          itemErrors[index].roleName = "Role is required";
        } else if (!availableRoleNames.has(roleName)) {
          itemErrors[index].roleName = "Selected role was not found in this organization";
        } else if (seenRoles.has(roleName)) {
          itemErrors[index].roleName = "Duplicate role approver is not allowed";
        } else {
          seenRoles.add(roleName);
        }
        break;
      }

      default: {
        itemErrors[index].type = "Unsupported approver type";
        return;
      }
    }
  });

  return { formErrors, itemErrors };
}
