import { Fieldset, Label } from "@/components/Fieldset/fieldset";
import { LoadingButton } from "@/components/ui/loading-button";
import { Switch } from "@/ui/switch";
import { Checkbox } from "@/ui/checkbox";
import { createOauthRestrictChangeHandler } from "../invitationOauth";

export interface OAuthInvitationSettingsPanelProps {
  policySummary: string;
  oauthHasUnsavedChanges: boolean;
  oauthRestrictProviders: boolean;
  setOauthRestrictProviders: (value: boolean) => void;
  oauthGithub: boolean;
  setOauthGithub: (value: boolean) => void;
  oauthGoogle: boolean;
  setOauthGoogle: (value: boolean) => void;
  setOauthSelectionError: (value: string | null) => void;
  oauthSelectionError: string | null;
  oauthMessage: string | null;
  canUpdateOrg: boolean;
  isPending: boolean;
  onSave: () => void;
}

function OAuthAllowedProviderCheckboxes({
  oauthGithub,
  oauthGoogle,
  setOauthGithub,
  setOauthGoogle,
  setOauthSelectionError,
  canUpdateOrg,
  isPending,
}: {
  oauthGithub: boolean;
  oauthGoogle: boolean;
  setOauthGithub: (value: boolean) => void;
  setOauthGoogle: (value: boolean) => void;
  setOauthSelectionError: (value: string | null) => void;
  canUpdateOrg: boolean;
  isPending: boolean;
}) {
  return (
    <div className="mt-4 border-t border-gray-200 pt-4 dark:border-gray-700">
      <p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Allowed OAuth providers</p>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-6">
        <div className="flex items-center gap-2">
          <Checkbox
            id="org-oauth-github"
            checked={oauthGithub}
            onCheckedChange={(checked) => {
              setOauthGithub(checked === true);
              setOauthSelectionError(null);
            }}
            disabled={!canUpdateOrg || isPending}
          />
          <Label htmlFor="org-oauth-github" className="text-sm font-normal cursor-pointer">
            GitHub
          </Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="org-oauth-google"
            checked={oauthGoogle}
            onCheckedChange={(checked) => {
              setOauthGoogle(checked === true);
              setOauthSelectionError(null);
            }}
            disabled={!canUpdateOrg || isPending}
          />
          <Label htmlFor="org-oauth-google" className="text-sm font-normal cursor-pointer">
            Google
          </Label>
        </div>
      </div>
    </div>
  );
}

export function OAuthInvitationSettingsPanel({
  policySummary,
  oauthHasUnsavedChanges,
  oauthRestrictProviders,
  setOauthRestrictProviders,
  oauthGithub,
  setOauthGithub,
  oauthGoogle,
  setOauthGoogle,
  setOauthSelectionError,
  oauthSelectionError,
  oauthMessage,
  canUpdateOrg,
  isPending,
  onSave,
}: OAuthInvitationSettingsPanelProps) {
  const onRestrictChange = createOauthRestrictChangeHandler({
    oauthGithub,
    oauthGoogle,
    setOauthRestrictProviders,
    setOauthGithub,
    setOauthGoogle,
    setOauthSelectionError,
  });

  return (
    <Fieldset
      className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6"
      data-testid="oauth-invitation-settings-card"
    >
      <div className="flex items-start justify-between gap-6">
        <div>
          <Label
            htmlFor="organization-oauth-invite-restrict-switch"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
          >
            OAuth providers for email invitations
          </Label>
          <p className="text-sm text-gray-500 dark:text-gray-400 max-w-prose">
            Applies when someone signs in with GitHub or Google and has a pending email invitation to this organization.
            Turn the switch on to allow only the providers you select; turn it off to allow any configured OAuth
            provider.
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-prose">
            Continue with email and password sign-in (when your installation allows them) are configured in the section
            below.
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">{policySummary}</p>
          {oauthHasUnsavedChanges ? (
            <p className="text-xs text-amber-800 dark:text-amber-400 mt-2">
              You have unsaved changes. Click Save OAuth settings to apply them.
            </p>
          ) : null}
        </div>
        <div className="flex items-center gap-3 shrink-0">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            {oauthRestrictProviders ? "Restricted" : "Any provider"}
          </span>
          <Switch
            id="organization-oauth-invite-restrict-switch"
            checked={oauthRestrictProviders}
            onCheckedChange={onRestrictChange}
            disabled={isPending || !canUpdateOrg}
            aria-label="Restrict OAuth providers for pending email invitations"
          />
        </div>
      </div>

      {oauthRestrictProviders ? (
        <OAuthAllowedProviderCheckboxes
          oauthGithub={oauthGithub}
          oauthGoogle={oauthGoogle}
          setOauthGithub={setOauthGithub}
          setOauthGoogle={setOauthGoogle}
          setOauthSelectionError={setOauthSelectionError}
          canUpdateOrg={canUpdateOrg}
          isPending={isPending}
        />
      ) : null}

      {oauthSelectionError ? <p className="mt-3 text-sm text-red-600">{oauthSelectionError}</p> : null}

      <div className="mt-4 flex items-center gap-4">
        <LoadingButton
          type="button"
          onClick={onSave}
          disabled={!canUpdateOrg}
          loading={isPending}
          loadingText="Saving..."
          className="max-w-48"
        >
          Save OAuth settings
        </LoadingButton>
        {oauthMessage ? (
          <span className={`text-sm ${oauthMessage.includes("Failed") ? "text-red-600" : "text-green-600"}`}>
            {oauthMessage}
          </span>
        ) : null}
      </div>
    </Fieldset>
  );
}
