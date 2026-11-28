import { Circle, CircleCheck, Trash2 } from "lucide-react";

import type { SsoConnection, SsoDomain } from "@/api-client/types.gen";
import { Badge } from "@/components/Badge/badge";
import { Button } from "@/components/ui/button";

import { FactorySettingsCard } from "../FactorySettingsCard";
import { SsoSelect } from "./SsoSettingsControls";
import { ActionRow, CardSave, CopyField, NeedsConnection, PermissionTip, TextField } from "./ssoSettingsFields";
import { testResultText, type SsoDraft, type SsoSettingsActions, type SsoSettingsModel } from "./ssoSettingsTypes";

const PRESETS = [
  { value: "PRESET_OKTA", label: "Okta" },
  { value: "PRESET_ENTRA", label: "Microsoft Entra ID" },
  { value: "PRESET_GENERIC", label: "Generic OIDC" },
];

export function SetupCard({ draft }: { draft: SsoDraft }) {
  const connected = draft.exists && draft.providerReady;
  return (
    <FactorySettingsCard title="Setup" data-testid="sso-setup">
      <ul className="space-y-2">
        <SetupRow done={draft.verified} label="Verify an email domain" />
        <SetupRow done={connected} label="Connect the identity provider" />
        <SetupRow done={draft.tested} label="Pass a connection test" />
      </ul>
      <p className="mt-3 text-[12px] leading-5 text-muted-foreground">
        Members see Single Sign-On on the login page after you activate the connection.
      </p>
    </FactorySettingsCard>
  );
}

function SetupRow({ done, label }: { done: boolean; label: string }) {
  const Icon = done ? CircleCheck : Circle;
  return (
    <li className="flex items-center gap-2 text-[13px]">
      <Icon className={done ? "size-4 text-foreground" : "size-4 text-muted-foreground"} aria-hidden />
      <span className={done ? "text-foreground" : "text-muted-foreground"}>{label}</span>
    </li>
  );
}

export function DomainCard({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  return (
    <FactorySettingsCard
      title="Email domains"
      description="Members with an email address on a verified domain can sign in with Single Sign-On."
      attachedList={model.domains.length > 0}
      data-testid="sso-domains"
    >
      <form
        className="flex items-end gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          if (draft.domain.trim()) actions.onAddDomain(draft.domain.trim());
        }}
      >
        <TextField
          id="sso-domain"
          label="Domain"
          value={draft.domain}
          placeholder="example.com"
          disabled={!draft.canChange}
          wrapperClassName="min-w-0 flex-1"
          onChange={draft.setDomain}
        />
        <PermissionTip permitted={draft.permitted}>
          <Button type="submit" className="shrink-0" disabled={!draft.canChange || !draft.domain.trim()}>
            Add domain
          </Button>
        </PermissionTip>
      </form>
      {model.domains.length > 0 ? (
        <ul className="mt-4 divide-y divide-border border-t border-border">
          {model.domains.map((item) => (
            <DomainRow key={item.id} item={item} model={model} actions={actions} draft={draft} />
          ))}
        </ul>
      ) : null}
    </FactorySettingsCard>
  );
}

function DomainRow({
  item,
  model,
  actions,
  draft,
}: {
  item: SsoDomain;
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  const verified = Boolean(item.verifiedAt);
  const domainError = model.domainError;
  const error = domainError && domainError.id === item.id ? domainError.message : null;
  return (
    <li className="space-y-3 py-3" data-testid={`sso-domain-${item.domain}`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-[13px] font-medium text-foreground">{item.domain}</span>
          <Badge color={verified ? "green" : "yellow"}>{verified ? "Verified" : "Pending"}</Badge>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {verified ? null : (
            <Button
              type="button"
              size="sm"
              disabled={!draft.canChange}
              onClick={() => item.id && actions.onVerifyDomain(item.id)}
            >
              Verify
            </Button>
          )}
          <PermissionTip permitted={draft.permitted}>
            <Button
              type="button"
              size="icon"
              variant="ghost"
              className="size-8 text-muted-foreground"
              disabled={!draft.canUpdate}
              aria-label={`Delete ${item.domain}`}
              onClick={() =>
                item.id &&
                draft.ask("Delete domain", "Delete this domain. This action cannot be undone.", "Delete domain", () =>
                  actions.onDeleteDomain(item.id!),
                )
              }
            >
              <Trash2 className="size-3.5" strokeWidth={1.75} aria-hidden />
            </Button>
          </PermissionTip>
        </div>
      </div>
      {verified ? null : <PendingDomainRecord item={item} />}
      {error ? <p className="text-[11px] text-destructive">{error}</p> : null}
    </li>
  );
}

function PendingDomainRecord({ item }: { item: SsoDomain }) {
  return (
    <div className="space-y-3">
      <p className="text-[12px] leading-5 text-muted-foreground">
        Add this TXT record at your DNS provider. Then select Verify.
      </p>
      <p className="text-[12px] text-muted-foreground">Type: TXT</p>
      <CopyField label="Name" value={item.txtName ?? ""} />
      <CopyField label="Value" value={item.txtValue ?? ""} />
    </div>
  );
}

export function ProviderCard({ model, draft }: { model: SsoSettingsModel; draft: SsoDraft }) {
  return (
    <FactorySettingsCard title="Identity provider" data-testid="sso-provider">
      <div className="space-y-6">
        <ProviderIdentity draft={draft} />
        {draft.exists ? (
          <CardSave
            permitted={draft.permitted}
            disabled={!draft.canChange || !draft.identityDirty}
            busy={Boolean(model.busy)}
            onClick={draft.saveProvider}
            testId="sso-identity-save"
          />
        ) : (
          <CreateConnection draft={draft} />
        )}
      </div>
    </FactorySettingsCard>
  );
}

export function ProviderUrlsCard({ model, draft }: { model: SsoSettingsModel; draft: SsoDraft }) {
  return (
    <FactorySettingsCard
      title="Values for your identity provider"
      description="Add these values to the application in your identity provider."
      data-testid="sso-provider-urls"
    >
      <ProviderUrls connection={model.connection} preset={draft.preset} />
    </FactorySettingsCard>
  );
}

export function ProviderValuesCard({ model, draft }: { model: SsoSettingsModel; draft: SsoDraft }) {
  return (
    <FactorySettingsCard
      title="Values from your identity provider"
      description="Copy these values from the application in your identity provider."
      data-testid="sso-provider-values"
    >
      <div className="space-y-6">
        <TextField
          id="sso-issuer"
          label="Issuer URL"
          value={draft.issuer}
          placeholder="https://example.okta.com"
          disabled={!draft.canChange}
          onChange={draft.setIssuer}
        />
        <TextField
          id="sso-client-id"
          label="Client ID"
          value={draft.clientId}
          disabled={!draft.canChange}
          onChange={draft.setClientId}
        />
        <TextField
          id="sso-client-secret"
          label="Client secret"
          type="password"
          value={draft.clientSecret}
          disabled={!draft.canChange}
          helper={
            model.connection?.clientSecretSet ? "A client secret is saved. Enter a new one to replace it." : undefined
          }
          onChange={draft.setClientSecret}
        />
        <CardSave
          permitted={draft.permitted}
          disabled={!draft.canChange || !draft.credentialsDirty}
          busy={Boolean(model.busy)}
          onClick={draft.saveProvider}
          testId="sso-provider-save"
        />
      </div>
    </FactorySettingsCard>
  );
}

function ProviderIdentity({ draft }: { draft: SsoDraft }) {
  return (
    <>
      <SsoSelect
        id="sso-preset"
        label="Provider"
        value={draft.preset}
        disabled={!draft.canChange}
        onChange={draft.setPreset}
        options={PRESETS}
      />
      <TextField
        id="sso-display-name"
        label="Display name"
        value={draft.displayName}
        disabled={!draft.canChange}
        helper="Members see this name on the login page."
        onChange={draft.setDisplayName}
      />
    </>
  );
}

function CreateConnection({ draft }: { draft: SsoDraft }) {
  return (
    <div className="space-y-2">
      <PermissionTip permitted={draft.permitted}>
        <Button type="button" disabled={!draft.canChange || !draft.displayName.trim()} onClick={draft.createConnection}>
          Create connection
        </Button>
      </PermissionTip>
      <p className="text-[12px] text-muted-foreground">
        SuperPlane then shows the URLs to add to your identity provider.
      </p>
    </div>
  );
}

function ProviderUrls({ connection, preset }: { connection?: SsoConnection | null; preset: string }) {
  const okta = preset === "PRESET_OKTA";
  return (
    <div className="space-y-6">
      <CopyField
        label="Redirect URI"
        value={connection?.redirectUri ?? ""}
        helper={okta ? "In Okta: Sign-in redirect URIs." : undefined}
      />
      <CopyField
        label="Initiate login URI"
        value={connection?.initiateLoginUri ?? ""}
        helper={okta ? "In Okta: Initiate login URI." : undefined}
      />
      <CopyField
        label="Post-logout redirect URI"
        value={connection?.postLogoutRedirectUri ?? ""}
        helper={okta ? "In Okta: Sign-out redirect URIs." : undefined}
      />
      {okta ? null : <CopyField label="Back-channel logout URI" value={connection?.backChannelLogoutUri ?? ""} />}
      <CopyField
        label="Global token revocation URL"
        value={connection?.globalTokenRevocationUri ?? ""}
        helper={
          okta
            ? "In Okta: Logout, API based logout setup, Endpoint URL. Select Signed JWT and Issuer and Subject Identifier."
            : undefined
        }
      />
    </div>
  );
}

export function ActivationCard({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  return (
    <FactorySettingsCard title="Test and activate" data-testid="sso-activation">
      {draft.exists ? <ActivationRows model={model} actions={actions} draft={draft} /> : <NeedsConnection />}
    </FactorySettingsCard>
  );
}

function ActivationRows({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  const result = testResultText(model.connection);
  const missingCredentials = !draft.providerReady;
  const testFailure = model.testFailure && !result?.includes(model.testFailure) ? model.testFailure : null;
  return (
    <div className="space-y-6">
      <ActionRow
        title="Test the connection"
        description="Sign in through the identity provider in a new window."
        detail={
          <>
            {result ? <p className="text-[12px] leading-5 text-muted-foreground">{result}</p> : null}
            {!result && !missingCredentials && model.connection?.status === "STATUS_ACTIVE" ? (
              <p className="text-[12px] leading-5 text-muted-foreground">
                This connection has no current test result. Test the connection again.
              </p>
            ) : null}
            {testFailure ? <p className="text-[12px] leading-5 text-muted-foreground">{testFailure}</p> : null}
            {missingCredentials ? (
              <p className="text-[12px] leading-5 text-muted-foreground">
                Add the issuer URL, client ID and client secret first.
              </p>
            ) : null}
          </>
        }
      >
        <PermissionTip permitted={draft.permitted}>
          <Button type="button" size="sm" disabled={!draft.canUpdate || missingCredentials} onClick={actions.onTest}>
            Test connection
          </Button>
        </PermissionTip>
      </ActionRow>
      {model.connection?.status === "STATUS_ACTIVE" ? (
        <DisableRow actions={actions} draft={draft} />
      ) : (
        <ActivateRow actions={actions} draft={draft} />
      )}
    </div>
  );
}

function ActivateRow({ actions, draft }: { actions: SsoSettingsActions; draft: SsoDraft }) {
  const missing = !draft.verified
    ? "Verify an email domain first."
    : !draft.tested
      ? "Pass a connection test first."
      : null;
  return (
    <ActionRow
      title="Activate"
      description="Members see Single Sign-On on the login page."
      detail={missing ? <p className="text-[12px] leading-5 text-muted-foreground">{missing}</p> : null}
    >
      <PermissionTip permitted={draft.permitted}>
        <Button type="button" size="sm" disabled={!draft.canChange || Boolean(missing)} onClick={actions.onActivate}>
          Activate
        </Button>
      </PermissionTip>
    </ActionRow>
  );
}

function DisableRow({ actions, draft }: { actions: SsoSettingsActions; draft: SsoDraft }) {
  return (
    <ActionRow title="Disable" description="Members sign in with other methods again.">
      <PermissionTip permitted={draft.permitted}>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={!draft.canUpdate}
          onClick={() =>
            draft.ask(
              "Disable Single Sign-On",
              "Members can sign in with other methods again.",
              "Disable Single Sign-On",
              actions.onDisable,
            )
          }
        >
          Disable
        </Button>
      </PermissionTip>
    </ActionRow>
  );
}
