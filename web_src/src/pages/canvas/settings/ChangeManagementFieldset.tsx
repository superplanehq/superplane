import { Fieldset, Label } from "@/components/Fieldset/fieldset";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { ApproversEditor } from "./ApproversEditor";
import type { ApproverValidationResult, ChangeRequestApproverType, SettingsApprover } from "./types";

const CHANGE_MANAGEMENT_ENFORCED_TOOLTIP =
  "Change management is enabled by your organization settings for all canvases.";

type ChangeManagementFieldsetProps = {
  isChangeManagementEnforcedByOrganization: boolean;
  changeManagementEnabled: boolean;
  onChangeManagementEnabledChange: (value: boolean) => void;
  isChangeManagementToggleDisabled: boolean;
  effectiveChangeManagementEnabled: boolean;
  approvers: SettingsApprover[];
  canUpdateCanvas: boolean;
  availableUsers: Array<{ id: string; name: string }>;
  availableRoles: Array<{ name: string; label: string }>;
  approverValidation: ApproverValidationResult;
  hasEveryoneApprover: boolean;
  onApproverTypeChange: (index: number, type: ChangeRequestApproverType) => void;
  onApproverUserChange: (index: number, userId: string) => void;
  onApproverRoleChange: (index: number, roleName: string) => void;
  onRemoveApprover: (index: number) => void;
  onAddApprover: () => void;
};

export function ChangeManagementFieldset({
  isChangeManagementEnforcedByOrganization,
  changeManagementEnabled,
  onChangeManagementEnabledChange,
  isChangeManagementToggleDisabled,
  effectiveChangeManagementEnabled,
  approvers,
  canUpdateCanvas,
  availableUsers,
  availableRoles,
  approverValidation,
  hasEveryoneApprover,
  onApproverTypeChange,
  onApproverUserChange,
  onApproverRoleChange,
  onRemoveApprover,
  onAddApprover,
}: ChangeManagementFieldsetProps) {
  const changeManagementContent = (
    <div className="flex items-start justify-between gap-6">
      <div>
        <Label htmlFor="canvas-change-management-switch" className="mb-1 block text-sm font-medium text-gray-700">
          Change Management
        </Label>
        <p className="text-[13px] text-gray-500">
          Require change requests with approvals before publishing canvas changes.
          {isChangeManagementEnforcedByOrganization
            ? " Change management is enabled by your organization settings for all canvases."
            : " This toggle controls change management for this canvas."}
        </p>
      </div>
      <div className="flex items-center gap-3">
        <span className="text-xs text-gray-500">
          {isChangeManagementEnforcedByOrganization ? "Enabled" : changeManagementEnabled ? "Enabled" : "Disabled"}
        </span>
        <Switch
          id="canvas-change-management-switch"
          checked={isChangeManagementEnforcedByOrganization ? true : changeManagementEnabled}
          onCheckedChange={onChangeManagementEnabledChange}
          disabled={isChangeManagementToggleDisabled}
          aria-label="Toggle canvas change management"
        />
      </div>
    </div>
  );

  return (
    <Fieldset className="space-y-6 border-t border-slate-200 pt-6">
      {isChangeManagementEnforcedByOrganization ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="cursor-not-allowed opacity-60">{changeManagementContent}</div>
          </TooltipTrigger>
          <TooltipContent side="top">{CHANGE_MANAGEMENT_ENFORCED_TOOLTIP}</TooltipContent>
        </Tooltip>
      ) : (
        changeManagementContent
      )}
      {effectiveChangeManagementEnabled ? (
        <ApproversEditor
          approvers={approvers}
          canUpdateCanvas={canUpdateCanvas}
          availableUsers={availableUsers}
          availableRoles={availableRoles}
          approverValidation={approverValidation}
          hasEveryoneApprover={hasEveryoneApprover}
          onApproverTypeChange={onApproverTypeChange}
          onApproverUserChange={onApproverUserChange}
          onApproverRoleChange={onApproverRoleChange}
          onRemoveApprover={onRemoveApprover}
          onAddApprover={onAddApprover}
        />
      ) : null}
    </Fieldset>
  );
}
