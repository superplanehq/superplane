import type { SsoscimToken } from "@/api-client/types.gen";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";

import { FactorySettingsCard } from "../FactorySettingsCard";
import { SsoSelect } from "./SsoSettingsControls";
import { CardSave, CopyField, NeedsConnection, PermissionTip, SwitchRow, TextField } from "./ssoSettingsFields";
import type { SsoDraft, SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

const ROLES = [
  { value: "org_admin", label: "Admin" },
  { value: "org_maintainer", label: "Maintainer" },
  { value: "org_operator", label: "Operator" },
];

export function ProvisioningCard({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  return (
    <FactorySettingsCard title="Provisioning" data-testid="sso-provisioning">
      {draft.exists ? <ProvisioningFields model={model} actions={actions} draft={draft} /> : <NeedsConnection />}
    </FactorySettingsCard>
  );
}

function ProvisioningFields({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  return (
    <div className="space-y-6">
      <SwitchRow
        id="sso-jit"
        title="Create members on first sign-in"
        description="A person with an email on a verified domain becomes a member with the default role."
        checked={draft.jitEnabled}
        disabled={!draft.canChange}
        onCheckedChange={draft.setJitEnabled}
      />
      <SsoSelect
        id="sso-role"
        label="Default role"
        value={draft.defaultRole}
        disabled={!draft.canChange}
        onChange={draft.setDefaultRole}
        options={ROLES}
      />
      <CardSave
        permitted={draft.permitted}
        disabled={!draft.canChange || !draft.provisioningDirty}
        busy={Boolean(model.busy)}
        onClick={draft.saveProvisioning}
        testId="sso-provisioning-save"
      />
      <Separator />
      <ScimGroup model={model} actions={actions} draft={draft} />
    </div>
  );
}

function ScimGroup({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  return (
    <div className="space-y-4">
      <p className="text-[13px] font-medium text-foreground">SCIM</p>
      <CopyField label="SCIM base URL" value={model.connection?.scimBaseUrl ?? ""} />
      {model.tokens.length > 0 ? (
        <ul className="divide-y divide-border border-t border-border">
          {model.tokens.map((token) => (
            <TokenRow key={token.id} token={token} actions={actions} draft={draft} />
          ))}
        </ul>
      ) : null}
      <TokenForm draft={draft} actions={actions} />
    </div>
  );
}

function TokenForm({ draft, actions }: { draft: SsoDraft; actions: SsoSettingsActions }) {
  return (
    <form
      className="flex flex-wrap items-end gap-2"
      onSubmit={(event) => {
        event.preventDefault();
        actions.onCreateToken(draft.tokenDescription);
      }}
    >
      <TextField
        id="sso-token-description"
        label="Token description"
        value={draft.tokenDescription}
        disabled={!draft.canChange}
        onChange={draft.setTokenDescription}
      />
      <PermissionTip permitted={draft.permitted}>
        <Button type="submit" disabled={!draft.canChange}>
          Create token
        </Button>
      </PermissionTip>
    </form>
  );
}

function TokenRow({ token, actions, draft }: { token: SsoscimToken; actions: SsoSettingsActions; draft: SsoDraft }) {
  const created = token.createdAt ? new Date(token.createdAt).toLocaleDateString() : null;
  return (
    <li className="flex flex-wrap items-center justify-between gap-3 py-2.5">
      <div className="min-w-0">
        <p className="truncate text-[13px] font-medium text-foreground">{token.description || "SCIM token"}</p>
        {created ? <p className="text-[12px] text-muted-foreground">{created}</p> : null}
      </div>
      <PermissionTip permitted={draft.permitted}>
        <RevokeToken token={token} actions={actions} draft={draft} />
      </PermissionTip>
    </li>
  );
}

function RevokeToken({ token, actions, draft }: { token: SsoscimToken; actions: SsoSettingsActions; draft: SsoDraft }) {
  return (
    <Button
      type="button"
      size="sm"
      variant="ghost"
      disabled={!draft.canUpdate}
      onClick={() => {
        if (!token.id) return;
        draft.ask("Revoke SCIM token", "The identity provider can no longer use this token.", "Revoke SCIM token", () =>
          actions.onRevokeToken(token.id!),
        );
      }}
    >
      Revoke
    </Button>
  );
}
