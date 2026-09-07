import { createContext, useContext, useState, type ReactNode } from "react";
import { Navigate, useLocation } from "react-router";
import { Toaster } from "sonner";

import { accountEmailOptions } from "@/lib/accountSettings";
import { showSuccessToast } from "@/lib/toast";

import { FactorySettingsLayout } from "../FactorySettingsLayout";
import { FactorySettingsNavProvider } from "../FactorySettingsNavProvider";
import { STORYBOOK_FACTORY_SETTINGS_NAV_GROUPS } from "../storybookFactorySettingsNav";
import { useFactories } from "@/hooks/useFactoryData";

import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import {
  ACCOUNT_REDESIGN_PROFILE,
  type AccountRedesignNotifications,
  type AccountRedesignProfile,
  type AccountRedesignSsoProvider,
} from "./accountProfileRedesignMocks";
import { AccountNotificationsRedesignPage } from "./AccountNotificationsRedesignPage";
import { AccountProfileRedesignPage } from "./AccountProfileRedesignPage";
import { AccountSecurityRedesignPage } from "./AccountSecurityRedesignPage";

interface AccountProfileRedesignState {
  profile: AccountRedesignProfile;
  setName: (name: string) => void;
  saveName: () => void;
  setEmail: (email: string) => void;
  connectSso: (provider: AccountRedesignSsoProvider) => void;
  disconnectSso: (provider: AccountRedesignSsoProvider) => void;
  changePassword: () => void;
  linkVelocityGithub: () => void;
  removeVelocityGithub: () => void;
  createToken: (name: string) => string;
  revokeToken: (id: string) => void;
  setNotifications: (notifications: AccountRedesignNotifications) => void;
  saveNotifications: () => void;
}

const AccountProfileRedesignContext = createContext<AccountProfileRedesignState | null>(null);

export function AccountProfileRedesignProvider({
  initialProfile = ACCOUNT_REDESIGN_PROFILE,
  children,
}: {
  initialProfile?: AccountRedesignProfile;
  children: ReactNode;
}) {
  const [profile, setProfile] = useState(initialProfile);

  const value: AccountProfileRedesignState = {
    profile,
    setName: (name) => setProfile((current) => ({ ...current, name })),
    saveName: () => setProfile((current) => ({ ...current, name: current.name.trim() })),
    setEmail: (email) => setProfile((current) => ({ ...current, email })),
    connectSso: (provider) => {
      setProfile((current) => ({
        ...current,
        ssoAccounts: current.ssoAccounts.map((account) =>
          account.provider === provider
            ? {
                ...account,
                identity: provider === "github" ? githubIdentity(current.name) : current.email,
                email: account.email || current.email,
              }
            : account,
        ),
      }));
      showSuccessToast(provider === "github" ? "GitHub connected." : "Google connected.");
    },
    disconnectSso: (provider) => {
      setProfile((current) => {
        const disconnected = current.ssoAccounts.find((account) => account.provider === provider);
        const ssoAccounts = current.ssoAccounts.map((account) =>
          account.provider === provider ? { ...account, identity: null, email: null } : account,
        );
        const nextEmail = nextEmailAfterDisconnect(current.email, disconnected?.email, ssoAccounts);
        return { ...current, email: nextEmail, ssoAccounts };
      });
      showSuccessToast(provider === "github" ? "GitHub disconnected." : "Google disconnected.");
    },
    changePassword: () => undefined,
    linkVelocityGithub: () => {
      setProfile((current) => ({
        ...current,
        velocityGithubUsername: githubIdentity(current.name),
      }));
      showSuccessToast("GitHub account linked.");
    },
    removeVelocityGithub: () => {
      setProfile((current) => ({ ...current, velocityGithubUsername: null }));
      showSuccessToast("GitHub link removed.");
    },
    createToken: (name) => {
      const id = `token-${profile.tokens.length + 1}`;
      const secret = `sp_pat_${id.replace("-", "")}_mock`;
      setProfile((current) => ({
        ...current,
        tokens: [{ id, name, createdAt: "Today" }, ...current.tokens],
      }));
      return secret;
    },
    revokeToken: (id) => {
      setProfile((current) => ({
        ...current,
        tokens: current.tokens.filter((token) => token.id !== id),
      }));
      showSuccessToast("Token revoked.");
    },
    setNotifications: (notifications) => setProfile((current) => ({ ...current, notifications })),
    saveNotifications: () => undefined,
  };

  return <AccountProfileRedesignContext.Provider value={value}>{children}</AccountProfileRedesignContext.Provider>;
}

function useAccountProfileRedesign() {
  const context = useContext(AccountProfileRedesignContext);
  if (!context) {
    throw new Error("Account redesign pages must render inside AccountProfileRedesignProvider");
  }
  return context;
}

export function AccountProfileRedesignRoutePage() {
  const {
    profile,
    setName,
    setEmail,
    saveName,
    linkVelocityGithub,
    removeVelocityGithub,
    changePassword,
    connectSso,
    disconnectSso,
    createToken,
    revokeToken,
  } = useAccountProfileRedesign();
  return (
    <AccountProfileRedesignPage
      name={profile.name}
      email={profile.email}
      emailOptions={accountEmailOptions({
        email: profile.email,
        hasPassword: profile.passwordSet,
        providers: profile.ssoAccounts.flatMap((account) =>
          account.identity && account.email ? [{ provider: account.provider, email: account.email }] : [],
        ),
      })}
      onNameChange={setName}
      onEmailChange={setEmail}
      onSave={saveName}
      velocityGithubUsername={profile.velocityGithubUsername}
      onLinkVelocityGithub={linkVelocityGithub}
      onRemoveVelocityGithub={removeVelocityGithub}
      security={
        <AccountSecurityRedesignPage
          passwordSet={profile.passwordSet}
          tokens={profile.tokens}
          ssoAccounts={profile.ssoAccounts}
          embedded
          onChangePassword={changePassword}
          onConnectSso={connectSso}
          onDisconnectSso={disconnectSso}
          onCreateToken={createToken}
          onRevokeToken={revokeToken}
        />
      }
    />
  );
}

export function AccountNotificationsRedesignRoutePage() {
  const { organizationId } = useFactorySettingsLayout();
  const { data: factories = [] } = useFactories(organizationId);
  const { profile, setNotifications, saveNotifications } = useAccountProfileRedesign();
  const workspaces = factories.flatMap((factory) =>
    factory.id && factory.name ? [{ id: factory.id, name: factory.name }] : [],
  );

  return (
    <AccountNotificationsRedesignPage
      email={profile.email}
      workspaces={workspaces}
      notifications={profile.notifications}
      onChange={setNotifications}
      onSave={saveNotifications}
    />
  );
}

export function AccountSecurityRedesignRoutePage() {
  const { profile, changePassword, connectSso, disconnectSso, createToken, revokeToken } = useAccountProfileRedesign();
  return (
    <AccountSecurityRedesignPage
      passwordSet={profile.passwordSet}
      tokens={profile.tokens}
      ssoAccounts={profile.ssoAccounts}
      onChangePassword={changePassword}
      onConnectSso={connectSso}
      onDisconnectSso={disconnectSso}
      onCreateToken={createToken}
      onRevokeToken={revokeToken}
    />
  );
}

/** Storybook factory settings chrome: redesign Profile, Security, and Notifications. */
export function StorybookAccountSettingsLayout({ initialProfile }: { initialProfile?: AccountRedesignProfile }) {
  return (
    <AccountProfileRedesignProvider initialProfile={initialProfile}>
      <FactorySettingsNavProvider groups={STORYBOOK_FACTORY_SETTINGS_NAV_GROUPS}>
        <FactorySettingsLayout />
      </FactorySettingsNavProvider>
      <Toaster position="bottom-center" closeButton />
    </AccountProfileRedesignProvider>
  );
}

export function StorybookAccountGeneralRedirect() {
  const { pathname, search } = useLocation();
  return <Navigate to={`${pathname.replace(/\/account\/general\/?$/, "/account/profile")}${search}`} replace />;
}

function nextEmailAfterDisconnect(
  currentEmail: string,
  disconnectedEmail: string | null | undefined,
  remaining: { identity: string | null; email?: string | null }[],
): string {
  if (!disconnectedEmail || disconnectedEmail.toLowerCase() !== currentEmail.toLowerCase()) {
    return currentEmail;
  }
  const next = remaining.find((account) => account.identity && account.email && account.email !== currentEmail);
  return next?.email || currentEmail;
}

function githubIdentity(name: string): string {
  const handle = name.trim().split(/\s+/)[0]?.toLowerCase();
  return handle || "user";
}
