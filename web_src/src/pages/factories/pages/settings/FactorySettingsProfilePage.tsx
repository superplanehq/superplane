import { useState } from "react";
import { Avatar } from "@/components/Avatar/avatar";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useAccount } from "@/contexts/useAccount";
import { useMe } from "@/hooks/useMe";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import { showErrorToast } from "@/lib/toast";
import { CopyButton } from "@/ui/CopyButton";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

function formatTokenDate(value?: string) {
  if (!value) return "Never";
  return new Date(value).toLocaleString();
}

export function FactorySettingsProfilePage() {
  const { organizationId } = useFactorySettingsLayout();
  const { data: user, isLoading, error: meError } = useMe();
  const { account } = useAccount();
  const tokensPanel = usePersonalTokensPanel(organizationId);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);

  const errorMessage = meError instanceof Error ? meError.message : meError ? "Failed to load profile" : null;
  const canChangePassword = account?.has_password === true;

  return (
    <>
      <FactorySettingsPageFrame title="General" subtitle="Update your personal account information and preferences.">
        <ProfileBody
          isLoading={isLoading}
          errorMessage={errorMessage}
          user={user}
          avatarUrl={account?.avatar_url}
          canChangePassword={canChangePassword}
          tokensPanel={tokensPanel}
          onChangePassword={() => setPasswordModalOpen(true)}
        />
      </FactorySettingsPageFrame>
      {canChangePassword ? <ChangePasswordDialog open={passwordModalOpen} onOpenChange={setPasswordModalOpen} /> : null}
    </>
  );
}

interface ProfileUser {
  id?: string;
  name?: string;
  email?: string;
  createdAt?: string;
}

function ProfileBody({
  isLoading,
  errorMessage,
  user,
  avatarUrl,
  canChangePassword,
  tokensPanel,
  onChangePassword,
}: {
  isLoading: boolean;
  errorMessage: string | null;
  user: ProfileUser | null | undefined;
  avatarUrl?: string | null;
  canChangePassword: boolean;
  tokensPanel: ReturnType<typeof usePersonalTokensPanel>;
  onChangePassword: () => void;
}) {
  if (isLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading profile…</p>;
  }
  if (errorMessage) {
    return <p className="text-[13px] text-destructive">{errorMessage}</p>;
  }
  if (!user) {
    return <p className="text-[13px] text-muted-foreground">No user data available.</p>;
  }

  return (
    <div className="contents">
      <ProfileInformationCard
        user={user}
        avatarUrl={avatarUrl}
        canChangePassword={canChangePassword}
        onChangePassword={onChangePassword}
      />
      <ApiTokensCard panel={tokensPanel} />
    </div>
  );
}

function initialsFor(name: string): string {
  const parts = name
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0]?.toUpperCase() ?? "");
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0];
  }
  return `${parts[0]}${parts[parts.length - 1]}`;
}

function ProfileInformationCard({
  user,
  avatarUrl,
  canChangePassword,
  onChangePassword,
}: {
  user: ProfileUser;
  avatarUrl?: string | null;
  canChangePassword: boolean;
  onChangePassword: () => void;
}) {
  const displayName = user.name?.trim() || user.email || "User";

  return (
    <section className="space-y-6" data-testid="factory-settings-profile-form">
      <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Profile information</h2>
      <div className="flex items-center gap-4">
        <Avatar
          src={avatarUrl || undefined}
          initials={avatarUrl ? undefined : initialsFor(displayName)}
          alt={displayName}
          className="size-16"
        />
        <p className="text-[15px] font-medium text-foreground">{displayName}</p>
      </div>
      <dl className="space-y-4">
        <ProfileField label="Name" value={user.name?.trim() || "Not available"} />
        <ProfileField label="User ID" value={user.id ?? ""} />
        <ProfileField label="Email address" value={user.email ?? ""} />
        <ProfileField
          label="Member since"
          value={user.createdAt ? new Date(user.createdAt).toLocaleDateString() : "Not available"}
        />
      </dl>
      {canChangePassword ? (
        <Button
          type="button"
          onClick={onChangePassword}
          className="flex items-center gap-2"
          data-testid="change-password-button"
        >
          <Icon name="lock" />
          Change password
        </Button>
      ) : null}
    </section>
  );
}

function ApiTokensCard({ panel }: { panel: ReturnType<typeof usePersonalTokensPanel> }) {
  return (
    <FactorySettingsCard title="API tokens">
      <p className="text-[12px] text-muted-foreground">
        Use a personal API token to authenticate API requests to SuperPlane. Keep your tokens secure and do not share
        them.
      </p>

      {panel.actionError ? <p className="mt-2 text-[12px] text-destructive">{panel.actionError}</p> : null}

      <form
        className="mt-4 flex flex-wrap items-end gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          void panel.createToken();
        }}
      >
        <div className="w-full max-w-xs">
          <label htmlFor="factory-new-token-name" className="mb-1 block text-[12px] font-medium text-foreground">
            Token name
          </label>
          <Input
            id="factory-new-token-name"
            type="text"
            value={panel.newTokenName}
            onChange={(e) => panel.setNewTokenName(e.target.value)}
            placeholder="e.g., CI token"
            data-testid="user-token-create-name"
          />
        </div>
        <LoadingButton
          type="submit"
          disabled={!panel.newTokenName.trim()}
          loading={panel.isCreating}
          loadingText="Creating..."
          className="flex items-center gap-2"
          data-testid="user-token-create-submit"
        >
          <Icon name="plus" />
          Create token
        </LoadingButton>
      </form>

      {panel.revealedToken ? (
        <NewTokenReveal
          token={panel.revealedToken.plaintext}
          tokenVisible={panel.tokenVisible}
          onToggleTokenVisible={() => panel.setTokenVisible(!panel.tokenVisible)}
        />
      ) : null}

      {!panel.tokensLoading && panel.tokens.length === 0 ? (
        <div className="mt-4 flex items-center gap-2">
          <Icon name="key-round" className="text-muted-foreground" />
          <p className="text-[13px] text-muted-foreground">No API tokens yet</p>
        </div>
      ) : null}

      {panel.tokens.length > 0 ? (
        <div className="mt-4 divide-y divide-border" data-testid="user-token-list">
          {panel.tokens.map((tokenItem) => (
            <div
              key={tokenItem.id}
              className="flex items-center justify-between gap-4 py-3"
              data-testid="user-token-row"
            >
              <div className="min-w-0">
                <p className="truncate text-[13px] font-medium text-foreground">{tokenItem.name || "Unnamed"}</p>
                <p className="text-[12px] text-muted-foreground">
                  Created {formatTokenDate(tokenItem.createdAt)} · Last used {formatTokenDate(tokenItem.lastUsedAt)}
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => panel.revokeToken(tokenItem.id || "", tokenItem.name || "")}
                disabled={panel.revokingId === tokenItem.id}
                className="shrink-0 text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                data-testid="user-token-revoke-btn"
              >
                <Icon name="trash-2" size="sm" />
                Revoke
              </Button>
            </div>
          ))}
        </div>
      ) : null}
    </FactorySettingsCard>
  );
}

function NewTokenReveal({
  token,
  tokenVisible,
  onToggleTokenVisible,
}: {
  token: string;
  tokenVisible: boolean;
  onToggleTokenVisible: () => void;
}) {
  return (
    <div className="mt-4 space-y-3">
      <p className="text-[13px] font-medium text-foreground">New API token</p>
      <div className="flex items-center gap-2 ph-no-capture">
        <Input
          type={tokenVisible ? "text" : "password"}
          value={token}
          readOnly
          className="flex-1 font-mono text-sm"
          data-testid="user-token-reveal-value"
        />
        <Button variant="outline" onClick={onToggleTokenVisible} aria-label="Toggle token visibility">
          <Icon name={tokenVisible ? "eye-closed" : "eye"} />
        </Button>
        <CopyButton
          variant="button"
          text={token}
          onCopyError={() => showErrorToast("Failed to copy API token.")}
          data-testid="user-token-reveal-copy"
        >
          Copy
        </CopyButton>
      </div>
      <p className="rounded-md border border-border bg-muted/40 p-3 text-[12px] text-muted-foreground">
        This token is shown once. Copy and store it now. If you lose it, revoke it and create a new one.
      </p>
    </div>
  );
}

function ProfileField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[12px] font-medium text-foreground">{label}</dt>
      <dd className="mt-0.5 text-[13px] text-muted-foreground">{value}</dd>
    </div>
  );
}
