import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { meRegenerateToken } from "@/api-client/sdk.gen";
import { Avatar } from "@/components/Avatar/avatar";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useAccount } from "@/contexts/useAccount";
import { useMe, meKeys } from "@/hooks/useMe";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { showErrorToast } from "@/lib/toast";
import { CopyButton } from "@/ui/CopyButton";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactorySettingsProfilePage() {
  const { organizationId } = useFactorySettingsLayout();
  const queryClient = useQueryClient();
  const { data: user, isLoading, error: meError } = useMe();
  const { account } = useAccount();
  const [actionError, setActionError] = useState<string | null>(null);
  const [token, setToken] = useState("");
  const [tokenVisible, setTokenVisible] = useState(false);
  const [regeneratingToken, setRegeneratingToken] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);

  const errorMessage =
    actionError || (meError instanceof Error ? meError.message : meError ? "Failed to load profile" : null);
  const canChangePassword = account?.has_password === true;

  const handleRegenerateToken = async () => {
    try {
      setActionError(null);
      setRegeneratingToken(true);
      const response = await meRegenerateToken(withOrganizationHeader({ organizationId }));
      setToken(response.data.token || "");
      setTokenVisible(true);
      queryClient.invalidateQueries({ queryKey: meKeys.me(organizationId, true) });
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to regenerate token");
    } finally {
      setRegeneratingToken(false);
    }
  };

  return (
    <>
      <FactorySettingsPageFrame title="General" subtitle="Update your personal account information and preferences.">
        <ProfileBody
          isLoading={isLoading}
          errorMessage={errorMessage}
          user={user}
          canChangePassword={canChangePassword}
          token={token}
          tokenVisible={tokenVisible}
          regeneratingToken={regeneratingToken}
          onChangePassword={() => setPasswordModalOpen(true)}
          onRegenerateToken={() => void handleRegenerateToken()}
          onToggleTokenVisible={() => setTokenVisible((visible) => !visible)}
        />
      </FactorySettingsPageFrame>
      {canChangePassword ? <ChangePasswordDialog open={passwordModalOpen} onOpenChange={setPasswordModalOpen} /> : null}
    </>
  );
}

interface ProfileUser {
  id?: string;
  email?: string;
  createdAt?: string;
  hasToken?: boolean;
}

function ProfileBody({
  isLoading,
  errorMessage,
  user,
  canChangePassword,
  token,
  tokenVisible,
  regeneratingToken,
  onChangePassword,
  onRegenerateToken,
  onToggleTokenVisible,
}: {
  isLoading: boolean;
  errorMessage: string | null;
  user: ProfileUser | null | undefined;
  canChangePassword: boolean;
  token: string;
  tokenVisible: boolean;
  regeneratingToken: boolean;
  onChangePassword: () => void;
  onRegenerateToken: () => void;
  onToggleTokenVisible: () => void;
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
      <ProfileInformationCard user={user} canChangePassword={canChangePassword} onChangePassword={onChangePassword} />
      <ApiTokenCard
        hasToken={Boolean(user.hasToken)}
        token={token}
        tokenVisible={tokenVisible}
        regeneratingToken={regeneratingToken}
        onRegenerateToken={onRegenerateToken}
        onToggleTokenVisible={onToggleTokenVisible}
      />
    </div>
  );
}

function ProfileInformationCard({
  user,
  canChangePassword,
  onChangePassword,
}: {
  user: ProfileUser;
  canChangePassword: boolean;
  onChangePassword: () => void;
}) {
  return (
    <section className="space-y-6" data-testid="factory-settings-profile-form">
      <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Profile information</h2>
      <div className="flex items-center gap-4">
        <Avatar
          initials={user.email ? user.email.charAt(0).toUpperCase() : "U"}
          alt="User Avatar"
          className="size-16"
        />
        <p className="text-[15px] font-medium text-foreground">{user.email}</p>
      </div>
      <dl className="space-y-4">
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

function ApiTokenCard({
  hasToken,
  token,
  tokenVisible,
  regeneratingToken,
  onRegenerateToken,
  onToggleTokenVisible,
}: {
  hasToken: boolean;
  token: string;
  tokenVisible: boolean;
  regeneratingToken: boolean;
  onRegenerateToken: () => void;
  onToggleTokenVisible: () => void;
}) {
  return (
    <FactorySettingsCard title="API token">
      <p className="text-[12px] text-muted-foreground">
        Use this token to authenticate API requests to SuperPlane. Keep your token secure and do not share it.
      </p>
      {!hasToken ? (
        <div className="mt-4 flex items-center gap-2">
          <Icon name="key-round" className="text-muted-foreground" />
          <p className="text-[13px] text-muted-foreground">No API token generated</p>
        </div>
      ) : null}
      <div className="mt-4 flex flex-wrap items-center gap-4">
        <LoadingButton
          onClick={onRegenerateToken}
          loading={regeneratingToken}
          loadingText="Regenerating..."
          className="flex items-center gap-2"
        >
          <Icon name="refresh-ccw" />
          {hasToken ? "Regenerate Token" : "Generate Token"}
        </LoadingButton>
        {hasToken && !token ? (
          <p className="text-[12px] text-muted-foreground">
            Your current token is hidden for security. Generate a new token to view it.
          </p>
        ) : null}
      </div>
      {token ? (
        <NewTokenReveal token={token} tokenVisible={tokenVisible} onToggleTokenVisible={onToggleTokenVisible} />
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
        <Input type={tokenVisible ? "text" : "password"} value={token} readOnly className="flex-1 font-mono text-sm" />
        <Button variant="outline" onClick={onToggleTokenVisible} aria-label="Toggle token visibility">
          <Icon name={tokenVisible ? "eye-closed" : "eye"} />
        </Button>
        <CopyButton variant="button" text={token} onCopyError={() => showErrorToast("Failed to copy API token.")}>
          Copy
        </CopyButton>
      </div>
      <p className="rounded-md border border-border bg-muted/40 p-3 text-[12px] text-muted-foreground">
        This token is shown once. Copy and store it now. If you lose it, generate a new token.
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
