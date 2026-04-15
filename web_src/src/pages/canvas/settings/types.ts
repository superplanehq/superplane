export type ChangeRequestApproverType = "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE";

export type SettingsApprover = {
  type: ChangeRequestApproverType;
  userId?: string;
  roleName?: string;
};

export type SettingsValues = {
  name: string;
  description: string;
  changeManagementEnabled: boolean;
  changeRequestApprovalConfig?: {
    items?: SettingsApprover[];
  };
};

export type ApproverFieldErrors = {
  type?: string;
  userId?: string;
  roleName?: string;
};

export type ApproverValidationResult = {
  formErrors: string[];
  itemErrors: ApproverFieldErrors[];
};

export type SettingsSavePayload = {
  name: string;
  description: string;
  changeManagementEnabled?: boolean;
  changeRequestApprovalConfig?: {
    items?: Array<{ type: "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE"; userId?: string; roleName?: string }>;
  };
};

export interface SettingsViewProps {
  initialValues: SettingsValues;
  canUpdateCanvas: boolean;
  orgChangeManagementEnabled?: boolean;
  isSaving: boolean;
  availableUsers: Array<{ id: string; name: string }>;
  availableRoles: Array<{ name: string; label: string }>;
  onSave: (values: SettingsSavePayload) => Promise<void>;
  onBackToCanvas?: () => void;
}
