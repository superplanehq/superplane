import { Fieldset, Label } from "@/components/Fieldset/fieldset";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { ApproversEditor } from "./ApproversEditor";
import type { ApproverValidationResult, ChangeRequestApproverType, SettingsApprover } from "./types";

const VERSIONING_ENFORCED_TOOLTIP = "Versioning is enabled by your organization settings for all canvases.";

type VersioningFieldsetProps = {
  isVersioningEnforcedByOrganization: boolean;
  versioningEnabled: boolean;
  onVersioningEnabledChange: (value: boolean) => void;
  isVersioningToggleDisabled: boolean;
  effectiveCanvasVersioningEnabled: boolean;
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

export function VersioningFieldset({
  isVersioningEnforcedByOrganization,
  versioningEnabled,
  onVersioningEnabledChange,
  isVersioningToggleDisabled,
  effectiveCanvasVersioningEnabled,
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
}: VersioningFieldsetProps) {
  const versioningContent = (
    <div className="flex items-start justify-between gap-6">
      <div>
        <Label htmlFor="canvas-versioning-switch" className="mb-1 block text-sm font-medium text-gray-700">
          Canvas Versioning
        </Label>
        <p className="text-[13px] text-gray-500">
          Manage canvas edits with drafts and publish flow. When disabled, users edit the live canvas directly.
          {isVersioningEnforcedByOrganization
            ? " Versioning is enabled by your organization settings for all canvases."
            : " This toggle controls versioning for this canvas."}
        </p>
      </div>
      <div className="flex items-center gap-3">
        <span className="text-xs text-gray-500">
          {isVersioningEnforcedByOrganization ? "Enabled" : versioningEnabled ? "Enabled" : "Disabled"}
        </span>
        <Switch
          id="canvas-versioning-switch"
          checked={isVersioningEnforcedByOrganization ? true : versioningEnabled}
          onCheckedChange={onVersioningEnabledChange}
          disabled={isVersioningToggleDisabled}
          aria-label="Toggle canvas versioning"
        />
      </div>
    </div>
  );

  return (
    <Fieldset className="rounded-lg border border-slate-950/15 bg-white p-6">
      {isVersioningEnforcedByOrganization ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="cursor-not-allowed opacity-60">{versioningContent}</div>
          </TooltipTrigger>
          <TooltipContent side="top">{VERSIONING_ENFORCED_TOOLTIP}</TooltipContent>
        </Tooltip>
      ) : (
        versioningContent
      )}
      {effectiveCanvasVersioningEnabled ? (
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
