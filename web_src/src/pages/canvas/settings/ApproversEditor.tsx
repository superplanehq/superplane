import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { EMPTY_SELECT_VALUE } from "./approverUtils";
import type { ApproverValidationResult, ChangeRequestApproverType, SettingsApprover } from "./types";

type ApproversEditorProps = {
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

export function ApproversEditor({
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
}: ApproversEditorProps) {
  return (
    <div className="mt-6 border-t border-slate-950/10 pt-6 space-y-4">
      <div>
        <p className="mb-1 block text-sm font-medium text-gray-700">Who can approve changes</p>
        <p className="text-[13px] text-gray-500">Define who can approve or reject change requests for this canvas.</p>
      </div>

      {approverValidation.formErrors.map((error) => (
        <p key={error} className="text-xs text-red-600">
          {error}
        </p>
      ))}

      <div className="space-y-3">
        {approvers.map((approver, index) => (
          <div key={`approver-${index}`} className="border-b border-slate-950/10 py-3">
            <div className="grid gap-3 md:grid-cols-[auto_1fr_auto] md:items-start">
              <div className="w-full md:w-[12rem] md:justify-self-start">
                <Select
                  value={approver.type}
                  disabled={!canUpdateCanvas}
                  onValueChange={(value) => onApproverTypeChange(index, value as ChangeRequestApproverType)}
                >
                  <SelectTrigger className="h-9 w-full" aria-label="Request approval from">
                    <SelectValue placeholder="Select approver type" />
                  </SelectTrigger>
                  <SelectContent className="max-h-60">
                    <SelectItem value="TYPE_ANYONE">Everyone</SelectItem>
                    <SelectItem value="TYPE_USER">Specific user</SelectItem>
                    <SelectItem value="TYPE_ROLE">Role</SelectItem>
                  </SelectContent>
                </Select>
                {approverValidation.itemErrors[index]?.type ? (
                  <p className="mt-2 text-xs text-red-600">{approverValidation.itemErrors[index]?.type}</p>
                ) : null}
              </div>

              {approver.type === "TYPE_USER" ? (
                <div>
                  <Select
                    value={approver.userId || EMPTY_SELECT_VALUE}
                    disabled={!canUpdateCanvas}
                    onValueChange={(value) => onApproverUserChange(index, value === EMPTY_SELECT_VALUE ? "" : value)}
                  >
                    <SelectTrigger className="h-9 w-full" aria-label="User">
                      <SelectValue placeholder="Select a user…" />
                    </SelectTrigger>
                    <SelectContent className="max-h-60">
                      <SelectItem value={EMPTY_SELECT_VALUE}>Select a user…</SelectItem>
                      {availableUsers.map((user) => (
                        <SelectItem key={user.id} value={user.id}>
                          {user.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {approverValidation.itemErrors[index]?.userId ? (
                    <p className="mt-2 text-xs text-red-600">{approverValidation.itemErrors[index]?.userId}</p>
                  ) : null}
                </div>
              ) : approver.type === "TYPE_ROLE" ? (
                <div>
                  <Select
                    value={approver.roleName || EMPTY_SELECT_VALUE}
                    disabled={!canUpdateCanvas}
                    onValueChange={(value) => onApproverRoleChange(index, value === EMPTY_SELECT_VALUE ? "" : value)}
                  >
                    <SelectTrigger className="h-9 w-full" aria-label="Role">
                      <SelectValue placeholder="Select a role…" />
                    </SelectTrigger>
                    <SelectContent className="max-h-60">
                      <SelectItem value={EMPTY_SELECT_VALUE}>Select a role…</SelectItem>
                      {availableRoles.map((role) => (
                        <SelectItem key={role.name} value={role.name}>
                          {role.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {approverValidation.itemErrors[index]?.roleName ? (
                    <p className="mt-2 text-xs text-red-600">{approverValidation.itemErrors[index]?.roleName}</p>
                  ) : null}
                </div>
              ) : (
                <div className="self-center text-xs text-gray-500">Any authenticated user can approve.</div>
              )}

              <div className="flex h-full items-start gap-2">
                <Button
                  type="button"
                  variant="outline"
                  disabled={!canUpdateCanvas || approvers.length <= 1}
                  onClick={() => onRemoveApprover(index)}
                >
                  Remove
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>

      <Button
        type="button"
        variant="outline"
        disabled={!canUpdateCanvas || hasEveryoneApprover}
        onClick={onAddApprover}
      >
        Add Approver
      </Button>
    </div>
  );
}
