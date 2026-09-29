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
} from "./accountProfileRedesignMocks";
import { AccountNotificationsRedesignPage } from "./AccountNotificationsRedesignPage";
import { AccountProfileAssociatedAccountsCard } from "./AccountProfileAssociatedAccountsCard";
import { AccountProfileRedesignPage } from "./AccountProfileRedesignPage";
import { AccountSecurityRedesignPage } from "./AccountSecurityRedesignPage";

interface AccountProfileRedesignState {
  profile: AccountRedesignProfile;
  setName: (name: string) => void;
  saveName: () => void;
  setEmail: (email: string) => void;
  linkGithub: () => void;
  removeGithub: () => void;
  changePassword: () => void;
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
    linkGithub: () => {
      setProfile((current) => ({
        ...current,
        linkedGithubUsername: githubIdentity(current.name),
      }));
      showSuccessToast("GitHub account linked.");
    },
    removeGithub: () => {
      setProfile((current) => ({ ...current, linkedGithubUsername: null }));
      showSuccessToast("GitHub link removed.");
    },
    changePassword: () => undefined,
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
  const { profile, setName, setEmail, saveName, linkGithub, removeGithub, changePassword, createToken, revokeToken } =
    useAccountProfileRedesign();
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
      associatedAccounts={
        <AccountProfileAssociatedAccountsCard
          githubUsername={profile.linkedGithubUsername}
          onLinkGithub={linkGithub}
          onRemoveGithub={removeGithub}
        />
      }
      security={
        <AccountSecurityRedesignPage
          passwordSet={profile.passwordSet}
          tokens={profile.tokens}
          embedded
          onChangePassword={changePassword}
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
  const { profile, changePassword, createToken, revokeToken } = useAccountProfileRedesign();
  return (
    <AccountSecurityRedesignPage
      passwordSet={profile.passwordSet}
      tokens={profile.tokens}
      onChangePassword={changePassword}
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

function githubIdentity(name: string): string {
  return name.trim().toLowerCase().replace(/\s+/g, "-") || "github-user";
}
