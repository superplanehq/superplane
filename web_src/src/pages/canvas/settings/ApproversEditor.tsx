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

export function ApproversEditor(props: ApproversEditorProps) {
  return (
    <div className="mt-6 space-y-4 border-t border-slate-200 pt-6">
      <SectionHeader />
      <Errors errors={props.approverValidation.formErrors} />
      <ApproverList {...props} />
      <AddApproverButton {...props} />
    </div>
  );
}

function SectionHeader() {
  return (
    <div>
      <p className="mb-1 block text-sm font-medium text-gray-700">Who can approve changes</p>
      <p className="text-[13px] text-gray-500">Define who can approve or reject change requests for this canvas.</p>
    </div>
  );
}

function Errors({ errors }: { errors: string[] }) {
  return (
    <div>
      {errors.map((error) => (
        <p key={error} className="text-xs text-red-600">
          {error}
        </p>
      ))}
    </div>
  );
}

function AddApproverButton(props: ApproversEditorProps) {
  const disabled = !props.canUpdateCanvas || props.hasEveryoneApprover;

  return (
    <Button type="button" variant="outline" disabled={disabled} onClick={props.onAddApprover}>
      Add Approver
    </Button>
  );
}

function ApproverList(props: ApproversEditorProps) {
  return (
    <div className="space-y-3">
      {props.approvers.map((approver, index) => (
        <ApproverItem key={`approver-${index}`} approver={approver} {...props} index={index} />
      ))}
    </div>
  );
}

function ApproverItem(props: ApproversEditorProps & { index: number; approver: SettingsApprover }) {
  const { approver, approverValidation, onApproverTypeChange, onRemoveApprover, index } = props;

  return (
    <div key={`approver-${index}`} className="py-2">
      <div className="grid gap-3 md:grid-cols-[auto_1fr_auto] md:items-start">
        <div className="w-full md:w-[12rem] md:justify-self-start">
          <Select
            value={approver.type}
            disabled={!props.canUpdateCanvas}
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
          <UserApprover {...props} index={index} approver={approver} />
        ) : approver.type === "TYPE_ROLE" ? (
          <RoleApprover {...props} index={index} approver={approver} />
        ) : (
          <div className="self-center text-xs text-gray-500">Any authenticated user can approve.</div>
        )}

        <div className="flex h-full items-start gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={!props.canUpdateCanvas || props.approvers.length <= 1}
            onClick={() => onRemoveApprover(index)}
          >
            Remove
          </Button>
        </div>
      </div>
    </div>
  );
}

function UserApprover(props: ApproversEditorProps & { index: number; approver: SettingsApprover }) {
  const { approver, canUpdateCanvas, availableUsers, approverValidation, onApproverUserChange, index } = props;

  return (
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
  );
}

function RoleApprover(props: ApproversEditorProps & { index: number; approver: SettingsApprover }) {
  const { approver, canUpdateCanvas, availableRoles, approverValidation, onApproverRoleChange, index } = props;

  return (
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
  );
}
