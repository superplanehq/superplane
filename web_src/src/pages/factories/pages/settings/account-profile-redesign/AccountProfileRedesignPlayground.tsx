import { useState } from "react";
import { Toaster } from "sonner";

import { showSuccessToast } from "@/lib/toast";

import {
  ACCOUNT_REDESIGN_PROFILE,
  type AccountRedesignPageId,
  type AccountRedesignProfile,
} from "./accountProfileRedesignMocks";
import { AccountPreferencesRedesignPage } from "./AccountPreferencesRedesignPage";
import { AccountProfileRedesignPage } from "./AccountProfileRedesignPage";
import { AccountProfileRedesignShell } from "./AccountProfileRedesignShell";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import { AccountSecurityRedesignPage } from "./AccountSecurityRedesignPage";

/**
 * Storybook-only Account settings redesign. Local state. No API calls.
 */
export function AccountProfileRedesignPlayground({
  initialPage = "profile",
  initialProfile = ACCOUNT_REDESIGN_PROFILE,
}: {
  initialPage?: AccountRedesignPageId;
  initialProfile?: AccountRedesignProfile;
}) {
  const [page, setPage] = useState<AccountRedesignPageId>(initialPage);
  const [navQuery, setNavQuery] = useState("");
  const [profile, setProfile] = useState(initialProfile);

  return (
    <>
      <AccountProfileRedesignShell
        activePage={page}
        navQuery={navQuery}
        onNavQueryChange={setNavQuery}
        onSelectPage={setPage}
      >
        {page === "profile" ? (
          <AccountProfileRedesignPage
            name={profile.name}
            email={profile.email}
            userId={profile.userId}
            avatarUrl={profile.avatarUrl}
            onNameChange={(name) => setProfile((current) => ({ ...current, name }))}
            onAvatarChange={(avatarUrl) => setProfile((current) => ({ ...current, avatarUrl }))}
            onEmailChange={(email) => setProfile((current) => ({ ...current, email }))}
            onSave={() => {
              setProfile((current) => ({ ...current, name: current.name.trim() }));
            }}
            onLeaveWorkspace={() => undefined}
            onDeleteAccount={() => undefined}
          />
        ) : null}
        {page === "security" ? (
          <AccountSecurityRedesignPage
            passwordSet={profile.passwordSet}
            twoFactorEnabled={profile.twoFactorEnabled}
            tokens={profile.tokens}
            sessions={profile.sessions}
            onChangePassword={() => undefined}
            onEnableTwoFactor={() => setProfile((current) => ({ ...current, twoFactorEnabled: true }))}
            onCreateToken={(name) => {
              const id = `token-${profile.tokens.length + 1}`;
              const secret = `sp_pat_${id.replace("-", "")}_mock`;
              setProfile((current) => ({
                ...current,
                tokens: [{ id, name, createdAt: "Today" }, ...current.tokens],
              }));
              return secret;
            }}
            onRevokeToken={(id) => {
              setProfile((current) => ({
                ...current,
                tokens: current.tokens.filter((token) => token.id !== id),
              }));
              showSuccessToast("Token revoked.");
            }}
            onRevokeSession={(id) => {
              setProfile((current) => ({
                ...current,
                sessions: current.sessions.filter((session) => session.id !== id),
              }));
              showSuccessToast("Session revoked.");
            }}
            onRevokeOtherSessions={() => {
              setProfile((current) => ({
                ...current,
                sessions: current.sessions.filter((session) => session.isCurrent),
              }));
              showSuccessToast("Other sessions revoked.");
            }}
          />
        ) : null}
        {page === "notifications" ? <NotificationsStub onOpenPreferences={() => setPage("preferences")} /> : null}
        {page === "preferences" ? (
          <AccountPreferencesRedesignPage
            theme={profile.theme}
            timezone={profile.timezone}
            onThemeChange={(theme) => setProfile((current) => ({ ...current, theme }))}
            onTimezoneChange={(timezone) => setProfile((current) => ({ ...current, timezone }))}
          />
        ) : null}
      </AccountProfileRedesignShell>
      <Toaster position="bottom-center" closeButton />
    </>
  );
}

function NotificationsStub({ onOpenPreferences }: { onOpenPreferences: () => void }) {
  return (
    <FactorySettingsPageFrame
      title="Notifications"
      subtitle="This page already shipped. The mockup does not replace it."
    >
      <FactorySettingsCard>
        <p className="text-[13px] text-muted-foreground">
          Account notification settings stay on the current Notifications page. Theme and timezone move to Preferences.
        </p>
        <button
          type="button"
          className="mt-3 text-[13px] text-foreground underline-offset-2 hover:underline"
          onClick={onOpenPreferences}
        >
          Open Preferences
        </button>
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}
