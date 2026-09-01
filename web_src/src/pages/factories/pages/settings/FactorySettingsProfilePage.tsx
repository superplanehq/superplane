import { useState } from "react";
import { Avatar } from "@/components/Avatar/avatar";
import { GitHubAccountConnection } from "@/components/GitHubAccountConnection";
import { Icon } from "@/components/Icon";
import { PersonalApiTokenDialogs, PersonalApiTokensTable } from "@/components/PersonalApiTokens";
import { Button } from "@/components/ui/button";
import type { ConnectedAccountProvider } from "@/contexts/accountContextState";
import { useAccount } from "@/contexts/useAccount";
import { useMe } from "@/hooks/useMe";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import type { PersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

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
          providers={account?.providers ?? []}
          impersonating={account?.impersonation?.active === true}
          tokensPanel={tokensPanel}
          onChangePassword={() => setPasswordModalOpen(true)}
        />
      </FactorySettingsPageFrame>
      <PersonalApiTokenDialogs panel={tokensPanel} />
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
  providers,
  impersonating,
  tokensPanel,
  onChangePassword,
}: {
  isLoading: boolean;
  errorMessage: string | null;
  user: ProfileUser | null | undefined;
  avatarUrl?: string | null;
  canChangePassword: boolean;
  providers: ConnectedAccountProvider[];
  impersonating: boolean;
  tokensPanel: PersonalTokensPanel;
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
      <FactorySettingsCard title="Connected accounts">
        <p className="mb-4 text-[12px] text-muted-foreground">
          Connect your profiles to match your activity to your SuperPlane account.
        </p>
        <GitHubAccountConnection providers={providers} impersonating={impersonating} />
      </FactorySettingsCard>
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

function ApiTokensCard({ panel }: { panel: PersonalTokensPanel }) {
  return (
    <FactorySettingsCard
      title="API tokens"
      action={
        <Button size="sm" onClick={panel.openCreateDialog} data-testid="user-token-create-btn">
          <Icon name="plus" />
          Create token
        </Button>
      }
    >
      <p className="text-[12px] text-muted-foreground">
        Use a personal API token to authenticate API requests to SuperPlane. Keep your tokens secure and do not share
        them.
      </p>
      <PersonalApiTokensTable
        className="mt-4"
        tokens={panel.tokens}
        isLoading={panel.tokensLoading}
        onCreate={panel.openCreateDialog}
        onRevoke={panel.requestRevoke}
      />
    </FactorySettingsCard>
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
