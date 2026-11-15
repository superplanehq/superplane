import type { SsoConnection, SsoDomain } from "@/api-client/types.gen";
import { Badge } from "@/components/Badge/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CopyButton } from "@/ui/CopyButton";

import type { ConfirmRequest, SsoDraft, SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

const ROLES = [
  { value: "org_admin", label: "Admin" },
  { value: "org_maintainer", label: "Maintainer" },
  { value: "org_operator", label: "Operator" },
];

export function DomainSection({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  return (
    <section className="space-y-3">
      <h2 className="text-base font-medium">Domains</h2>
      <p className="text-sm text-gray-600">Add a domain. Publish the TXT record. Then verify the domain.</p>
      <form
        className="flex gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          if (draft.domain.trim()) actions.onAddDomain(draft.domain.trim());
        }}
      >
        <Label className="sr-only" htmlFor="sso-domain">
          Domain
        </Label>
        <Input
          id="sso-domain"
          value={draft.domain}
          onChange={(event) => draft.setDomain(event.target.value)}
          disabled={!draft.canChange}
        />
        <Button type="submit" disabled={!draft.canChange}>
          Add domain
        </Button>
      </form>
      {model.domains.map((item) => (
        <DomainRow key={item.id} item={item} actions={actions} draft={draft} />
      ))}
    </section>
  );
}

function DomainRow({ item, actions, draft }: { item: SsoDomain; actions: SsoSettingsActions; draft: SsoDraft }) {
  return (
    <div className="space-y-2" data-testid={`sso-domain-${item.domain}`}>
      <div className="flex items-center gap-2">
        <span>{item.domain}</span>
        <Badge>{item.verifiedAt ? "Verified" : "Pending"}</Badge>
      </div>
      <CopyField label="TXT name" value={item.txtName ?? ""} />
      <CopyField label="TXT value" value={item.txtValue ?? ""} />
      <Button
        type="button"
        disabled={!draft.canChange || Boolean(item.verifiedAt)}
        onClick={() => item.id && actions.onVerifyDomain(item.id)}
      >
        Verify
      </Button>
      <Button
        type="button"
        variant="outline"
        onClick={() =>
          item.id &&
          draft.ask("Delete domain", "Delete this domain. This action cannot be undone.", () =>
            actions.onDeleteDomain(item.id!),
          )
        }
      >
        Delete
      </Button>
    </div>
  );
}

export function ProviderSection({
  model,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  const connection = model.connection;
  return (
    <section className="space-y-3">
      <h2 className="text-base font-medium">Identity provider</h2>
      <Label htmlFor="sso-preset">Preset</Label>
      <select
        id="sso-preset"
        value={draft.preset}
        disabled={!draft.canChange}
        onChange={(event) => draft.setPreset(event.target.value)}
      >
        <option value="PRESET_OKTA">Okta</option>
        <option value="PRESET_ENTRA">Microsoft Entra ID</option>
        <option value="PRESET_GENERIC">Generic OIDC</option>
      </select>
      <Label htmlFor="sso-display-name">Display name</Label>
      <Input
        id="sso-display-name"
        value={draft.displayName}
        disabled={!draft.canChange}
        onChange={(event) => draft.setDisplayName(event.target.value)}
      />
      {draft.exists ? <ProviderUrls connection={connection} /> : null}
      <Label htmlFor="sso-issuer">Issuer URL</Label>
      <Input
        id="sso-issuer"
        value={draft.issuer}
        disabled={!draft.canChange}
        onChange={(event) => draft.setIssuer(event.target.value)}
      />
      <Label htmlFor="sso-client-id">Client ID</Label>
      <Input
        id="sso-client-id"
        value={draft.clientId}
        disabled={!draft.canChange}
        onChange={(event) => draft.setClientId(event.target.value)}
      />
      <Label htmlFor="sso-client-secret">Client secret</Label>
      <Input
        id="sso-client-secret"
        type="password"
        value={draft.clientSecret}
        disabled={!draft.canChange}
        onChange={(event) => draft.setClientSecret(event.target.value)}
      />
      <p>{connection?.clientSecretSet ? "Saved" : "Client secret is not saved."}</p>
      <Button type="button" disabled={!draft.canChange} onClick={() => draft.save()}>
        Save
      </Button>
    </section>
  );
}

function ProviderUrls({ connection }: { connection?: SsoConnection | null }) {
  return (
    <>
      <CopyField label="Redirect URI" value={connection?.redirectUri ?? ""} />
      <CopyField label="Initiate login URI" value={connection?.initiateLoginUri ?? ""} />
      <CopyField label="Post-logout redirect URI" value={connection?.postLogoutRedirectUri ?? ""} />
      <CopyField label="Back-channel logout URI" value={connection?.backChannelLogoutUri ?? ""} />
    </>
  );
}

export function ActivationSection({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  const connection = model.connection;
  return (
    <section className="space-y-3" data-testid="sso-activation">
      <h2 className="text-base font-medium">Test and activate</h2>
      <p>Status: {statusLabel(connection?.status)}</p>
      <p data-testid="sso-test-result">
        {connection?.lastTest ? (draft.tested ? "Last test succeeded." : "Last test failed.") : "No test yet."}{" "}
        {connection?.lastTest?.email}
      </p>
      <div className="flex flex-wrap gap-2">
        <Button type="button" disabled={!draft.exists} onClick={actions.onTest}>
          Test connection
        </Button>
        <Button
          type="button"
          disabled={!draft.canChange || !draft.tested || !draft.verified}
          onClick={actions.onActivate}
        >
          Activate
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={!draft.exists}
          onClick={() =>
            draft.ask("Disable Single Sign-On", "Members can sign in with other methods again.", actions.onDisable)
          }
        >
          Disable
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={!draft.exists}
          onClick={() =>
            draft.ask("Delete Single Sign-On", "Delete the connection and its identities.", actions.onDelete)
          }
        >
          Delete
        </Button>
      </div>
    </section>
  );
}

export function ProvisioningSection({
  model,
  actions,
  draft,
}: {
  model: SsoSettingsModel;
  actions: SsoSettingsActions;
  draft: SsoDraft;
}) {
  return (
    <section className="space-y-3">
      <h2 className="text-base font-medium">Provisioning</h2>
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={draft.jitEnabled}
          disabled={!draft.canChange}
          onChange={(event) => draft.setJitEnabled(event.target.checked)}
        />
        Create a member on the first sign-in
      </label>
      <Label htmlFor="sso-role">Default role</Label>
      <select
        id="sso-role"
        value={draft.defaultRole}
        disabled={!draft.canChange}
        onChange={(event) => draft.setDefaultRole(event.target.value)}
      >
        {ROLES.map((role) => (
          <option key={role.value} value={role.value}>
            {role.label}
          </option>
        ))}
      </select>
      <CopyField label="SCIM base URL" value={model.connection?.scimBaseUrl ?? ""} />
      <ul>
        {model.tokens.map((token) => (
          <li key={token.id} className="flex items-center gap-2">
            <span>{token.description || "SCIM token"}</span>
            <Button
              type="button"
              variant="outline"
              onClick={() =>
                token.id &&
                draft.ask("Revoke SCIM token", "The identity provider can no longer use this token.", () =>
                  actions.onRevokeToken(token.id!),
                )
              }
            >
              Revoke
            </Button>
          </li>
        ))}
      </ul>
      <div className="flex gap-2">
        <Input
          value={draft.tokenDescription}
          disabled={!draft.canChange || !draft.exists}
          onChange={(event) => draft.setTokenDescription(event.target.value)}
        />
        <Button
          type="button"
          disabled={!draft.canChange || !draft.exists}
          onClick={() => actions.onCreateToken(draft.tokenDescription)}
        >
          Create SCIM token
        </Button>
      </div>
    </section>
  );
}

export function PolicySection({ draft }: { draft: SsoDraft }) {
  return (
    <section className="space-y-3">
      <h2 className="text-base font-medium">Policy</h2>
      <Label htmlFor="sso-enforcement">Enforcement</Label>
      <select
        id="sso-enforcement"
        value={draft.enforcement}
        disabled={!draft.canChange}
        onChange={(event) => {
          const next = event.target.value;
          if (next === "ENFORCEMENT_REQUIRED") {
            draft.requestRequired();
            return;
          }
          draft.setEnforcement(next);
        }}
      >
        <option value="ENFORCEMENT_OPTIONAL">Optional</option>
        <option value="ENFORCEMENT_REQUIRED">Required</option>
      </select>
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={draft.ownerBypass}
          disabled={!draft.canChange}
          onChange={(event) => draft.setOwnerBypass(event.target.checked)}
        />
        Allow an owner to sign in with another method
      </label>
      <Label htmlFor="sso-session-hours">Maximum session age in hours</Label>
      <Input
        id="sso-session-hours"
        value={draft.sessionHours}
        disabled={!draft.canChange}
        onChange={(event) => draft.setSessionHours(event.target.value)}
      />
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={draft.idpLogout}
          disabled={!draft.canChange}
          onChange={(event) => draft.setIdpLogout(event.target.checked)}
        />
        Sign out from the identity provider
      </label>
      <Button type="button" disabled={!draft.canChange} onClick={() => draft.save()}>
        Save policy
      </Button>
    </section>
  );
}

export function ConfirmDialog({ confirm, onClose }: { confirm: ConfirmRequest | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(confirm)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{confirm?.title}</DialogTitle>
          <DialogDescription>{confirm?.body}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={() => {
              confirm?.run();
              onClose();
            }}
          >
            Confirm
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function RequiredDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Require Single Sign-On</DialogTitle>
          <DialogDescription>
            Members must sign in through the identity provider to open this organization. Owners can still use another
            method when owner bypass is on.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={onConfirm}>
            Require Single Sign-On
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function TokenDialog({ secret, onDismiss }: { secret?: string | null; onDismiss: () => void }) {
  return (
    <Dialog open={Boolean(secret)} onOpenChange={(open) => !open && onDismiss()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>SCIM token</DialogTitle>
          <DialogDescription>Copy this token now. You cannot see it again.</DialogDescription>
        </DialogHeader>
        <div className="flex items-center gap-2">
          <Input readOnly value={secret ?? ""} />
          <CopyButton variant="button" text={secret ?? ""} copiedLabel="Copied">
            Copy
          </CopyButton>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function CopyField({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2">
      <Label>{label}</Label>
      <Input readOnly value={value} />
      <CopyButton variant="button" text={value} copiedLabel="Copied">
        Copy
      </CopyButton>
    </div>
  );
}

function statusLabel(status: string | undefined) {
  if (status === "STATUS_ACTIVE") return "Active";
  if (status === "STATUS_DISABLED") return "Disabled";
  return "Draft";
}
