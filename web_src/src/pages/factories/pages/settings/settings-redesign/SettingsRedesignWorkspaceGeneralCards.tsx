import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { getNameInitials } from "@/lib/nameInitials";
import { Trash2 } from "lucide-react";

import { WORKSPACE_KEY_MAX_LENGTH } from "../../../lib/workspaceKey";
import {
  SettingsDangerPanel,
  SettingsIdentityHero,
  SettingsSaveBar,
  SettingsStackedField,
  SettingsUrlField,
} from "./settingsRedesignParts";
import {
  WORKSPACE_DESCRIPTION_MAX_LENGTH,
  WORKSPACE_NAME_MAX_LENGTH,
  type SettingsRedesignWorkspaceGeneral,
} from "./useSettingsRedesignWorkspaceGeneral";

export function WorkspaceIdentityForm({
  general,
  organizationSlug,
}: {
  general: SettingsRedesignWorkspaceGeneral;
  organizationSlug: string;
}) {
  const displayName = general.name.trim() || general.factory.name || "Workspace";
  const keyCaption = `/${organizationSlug}/workspaces/${general.key || "KEY"}`;

  return (
    <div className="space-y-6">
      <SettingsIdentityHero initials={getNameInitials(displayName)} title={displayName} caption={keyCaption} />

      <SettingsStackedField htmlFor="factory-settings-name" label="Name">
        <Input
          id="factory-settings-name"
          data-testid="factory-settings-name"
          value={general.name}
          onChange={(event) => {
            if (event.target.value.length <= WORKSPACE_NAME_MAX_LENGTH) {
              general.setName(event.target.value);
              if (general.nameError) general.clearNameError();
            }
          }}
          maxLength={WORKSPACE_NAME_MAX_LENGTH}
          disabled={!general.canUpdate}
        />
        {general.nameError ? <p className="text-[11px] text-destructive">{general.nameError}</p> : null}
      </SettingsStackedField>

      <SettingsStackedField
        htmlFor="factory-settings-key"
        label="Workspace key"
        hint="Changing the key updates every task identifier for this workspace."
      >
        <SettingsUrlField
          prefix={`/${organizationSlug}/workspaces/`}
          id="factory-settings-key"
          testId="factory-settings-key"
          value={general.key}
          disabled={!general.canUpdate}
          autoComplete="off"
          className="uppercase tracking-wider"
          onChange={(value) => {
            general.setKey(value.slice(0, WORKSPACE_KEY_MAX_LENGTH));
            if (general.keyError) general.clearKeyError();
          }}
        />
        {general.keyError ? <p className="text-[11px] text-destructive">{general.keyError}</p> : null}
      </SettingsStackedField>

      <SettingsStackedField htmlFor="factory-settings-description" label="Description">
        <Textarea
          id="factory-settings-description"
          data-testid="factory-settings-description"
          value={general.description}
          onChange={(event) => {
            if (event.target.value.length <= WORKSPACE_DESCRIPTION_MAX_LENGTH) {
              general.setDescription(event.target.value);
            }
          }}
          rows={3}
          disabled={!general.canUpdate}
        />
      </SettingsStackedField>

      <SettingsSaveBar
        allowed={general.canUpdate}
        permissionsLoading={general.permissionsLoading}
        disabled={!general.canUpdate || !general.name.trim() || !general.isDirty}
        loading={general.updateFactory.isPending}
        onSave={() => void general.saveDetails()}
        denyMessage="You do not have permission to update workspaces."
        testId="factory-settings-save"
      />
    </div>
  );
}

export function WorkspaceDangerSection({ general }: { general: SettingsRedesignWorkspaceGeneral }) {
  return (
    <SettingsDangerPanel
      testId="factory-settings-danger-zone"
      title="Delete workspace"
      description="Permanently removes tasks, lines, and apps."
      action={
        <PermissionTooltip
          allowed={general.canDelete || general.permissionsLoading}
          message="You do not have permission to delete workspaces."
        >
          <Button
            type="button"
            variant="destructive"
            size="sm"
            disabled={!general.canDelete}
            onClick={() => general.setDeleteOpen(true)}
            data-testid="factory-settings-delete-button"
          >
            <Trash2 className="size-3.5" aria-hidden />
            Delete workspace
          </Button>
        </PermissionTooltip>
      }
    />
  );
}
