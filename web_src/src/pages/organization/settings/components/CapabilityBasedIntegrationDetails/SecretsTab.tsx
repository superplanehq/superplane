import type { OrganizationsIntegrationSecret } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import type { Dispatch, SetStateAction } from "react";

export interface SecretsTabProps {
  integrationSecrets: OrganizationsIntegrationSecret[];
  secretDrafts: Record<string, string>;
  setSecretDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  settingsMutationBusy: boolean;
  saveSecret: (secretName: string, value: string, draftFieldKey: string) => Promise<void>;
  isSavingSecret: (secretName: string | undefined) => boolean;
}

export function SecretsTab({
  integrationSecrets,
  secretDrafts,
  setSecretDrafts,
  canUpdateIntegrations,
  permissionsLoading,
  settingsMutationBusy,
  saveSecret,
  isSavingSecret,
}: SecretsTabProps) {
  return (
    <>
      <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">
        Stored values are never shown. You can enter a replacement only when a secret is editable.
      </p>
      {integrationSecrets.length > 0 ? (
        <div className="space-y-6 rounded-lg border border-gray-300 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
          {integrationSecrets.map((secret, index) => {
            const fieldKey = secret.name?.trim() || `__secret_${index}`;
            const title = secret.label?.trim() || "Secret";
            const isEditable = secret.editable === true;
            const draft = secretDrafts[fieldKey] ?? "";
            const secretHasNewValue = draft.trim() !== "";
            const savingThisSecret = isSavingSecret(secret.name?.trim());

            return (
              <div
                key={fieldKey}
                className="border-b border-gray-200 pb-6 last:border-b-0 last:pb-0 dark:border-gray-800"
              >
                <div className="text-sm font-medium text-gray-800 dark:text-gray-100">{title}</div>
                {secret.description ? (
                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{secret.description}</p>
                ) : null}
                <div className="mt-3">
                  {isEditable ? (
                    <PermissionTooltip
                      allowed={canUpdateIntegrations || permissionsLoading}
                      message="You don't have permission to update integrations."
                    >
                      <div className="flex max-w-xl flex-col gap-2 sm:max-w-none sm:flex-row sm:items-center sm:gap-3">
                        <Input
                          type="password"
                          autoComplete="new-password"
                          value={draft}
                          placeholder="New secret value"
                          onChange={(event) =>
                            setSecretDrafts((previous) => ({
                              ...previous,
                              [fieldKey]: event.target.value,
                            }))
                          }
                          disabled={!canUpdateIntegrations || settingsMutationBusy}
                          className="w-full sm:max-w-xl sm:flex-1"
                        />
                        {secret.name?.trim() ? (
                          <LoadingButton
                            type="button"
                            color="blue"
                            size="sm"
                            className="shrink-0"
                            disabled={!canUpdateIntegrations || !secretHasNewValue || settingsMutationBusy}
                            loading={savingThisSecret}
                            loadingText="Saving…"
                            onClick={() => void saveSecret(secret.name!.trim(), draft, fieldKey)}
                          >
                            Save
                          </LoadingButton>
                        ) : null}
                      </div>
                    </PermissionTooltip>
                  ) : (
                    <p className="text-sm text-gray-500 dark:text-gray-400">This secret cannot be changed here.</p>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="text-sm text-gray-500 dark:text-gray-400">No secrets stored for this integration.</p>
      )}
    </>
  );
}
