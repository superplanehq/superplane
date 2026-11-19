import { Button } from "@/components/ui/button";

import { IntakeSettingsRadioOption } from "../../IntakeSettingsRadioOption";
import { FactorySettingsCard } from "../FactorySettingsCard";
import { ActionRow, CardSave, NeedsConnection, PermissionTip, SwitchRow, TextField } from "./ssoSettingsFields";
import type { SsoDraft, SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

export function PolicyCard({ model, draft }: { model: SsoSettingsModel; draft: SsoDraft }) {
  return (
    <FactorySettingsCard title="Sign-in policy" data-testid="sso-policy">
      {draft.exists ? <PolicyFields model={model} draft={draft} /> : <NeedsConnection />}
    </FactorySettingsCard>
  );
}

function PolicyFields({ model, draft }: { model: SsoSettingsModel; draft: SsoDraft }) {
  return (
    <div className="space-y-6">
      <EnforcementField draft={draft} />
      <SwitchRow
        id="sso-owner-bypass"
        title="Owner bypass"
        description="Owners can sign in with another method. Use it when the identity provider is not available."
        checked={draft.ownerBypass}
        disabled={!draft.canChange}
        onCheckedChange={draft.setOwnerBypass}
      />
      <TextField
        id="sso-session-hours"
        label="Maximum session age"
        value={draft.sessionHours}
        disabled={!draft.canChange}
        helper="Members sign in again after this time. Enter hours, such as 24 or 1.5, or minutes, such as 15m. From 15 minutes to 7 days."
        error={draft.sessionError}
        className="w-full max-w-[8rem]"
        onChange={draft.setSessionHours}
      />
      <SwitchRow
        id="sso-idp-logout"
        title="Sign out from the identity provider"
        description="When a member signs out of SuperPlane, the identity provider session also ends."
        checked={draft.idpLogout}
        disabled={!draft.canChange}
        onCheckedChange={draft.setIdpLogout}
      />
      <CardSave
        permitted={draft.permitted}
        disabled={!draft.canChange || !draft.policyDirty}
        busy={Boolean(model.busy)}
        onClick={draft.savePolicy}
        testId="sso-policy-save"
      />
    </div>
  );
}

function EnforcementField({ draft }: { draft: SsoDraft }) {
  return (
    <fieldset className="space-y-3">
      <legend className="text-[13px] font-medium text-foreground">Enforcement</legend>
      <IntakeSettingsRadioOption
        name="sso-enforcement"
        value="ENFORCEMENT_OPTIONAL"
        checked={draft.enforcement === "ENFORCEMENT_OPTIONAL"}
        title="Optional"
        helper="Members can sign in with Single Sign-On or another method."
        disabled={!draft.canChange}
        variant="plain"
        onChange={() => draft.setEnforcement("ENFORCEMENT_OPTIONAL")}
      />
      <IntakeSettingsRadioOption
        name="sso-enforcement"
        value="ENFORCEMENT_REQUIRED"
        checked={draft.enforcement === "ENFORCEMENT_REQUIRED"}
        title="Required"
        helper="Members must sign in with Single Sign-On to open this organization."
        disabled={!draft.canChange}
        variant="plain"
        onChange={draft.requestRequired}
      />
    </fieldset>
  );
}

export function DangerCard({ actions, draft }: { actions: SsoSettingsActions; draft: SsoDraft }) {
  return (
    <FactorySettingsCard title="Danger zone" data-testid="sso-danger">
      <ActionRow
        title="Delete connection"
        description="SuperPlane deletes the connection, its SCIM tokens and the links between members and the identity provider."
      >
        <PermissionTip permitted={draft.permitted}>
          <Button
            type="button"
            size="sm"
            variant="destructive"
            disabled={!draft.canUpdate}
            onClick={() =>
              draft.ask(
                "Delete connection",
                "Delete the connection and its identities.",
                "Delete connection",
                actions.onDelete,
              )
            }
          >
            Delete connection
          </Button>
        </PermissionTip>
      </ActionRow>
    </FactorySettingsCard>
  );
}
