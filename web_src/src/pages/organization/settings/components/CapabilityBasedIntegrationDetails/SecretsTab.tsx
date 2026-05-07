import type { OrganizationsIntegrationSecret } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Check, Pencil, X } from "lucide-react";
import { DescriptionTooltip } from "./DescriptionTooltip";
import type { Dispatch, SetStateAction } from "react";
import { useState } from "react";

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

const READONLY_SECRET_INPUT_CLASS =
  "cursor-default bg-gray-50 opacity-100 dark:bg-gray-950 [&::placeholder]:opacity-70";

type SecretEditingControlsProps = {
  inputId: string;
  fieldKey: string;
  draft: string;
  secretHasNewValue: boolean;
  savingThisSecret: boolean;
  secretNameTrimmed: string | undefined;
  canUpdateIntegrations: boolean;
  settingsMutationBusy: boolean;
  setSecretDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  setEditingFieldKey: Dispatch<SetStateAction<string | null>>;
  exitEdit: () => void;
  saveSecret: (secretName: string, value: string, draftFieldKey: string) => Promise<void>;
};

function SecretEditingControls({
  inputId,
  fieldKey,
  draft,
  secretHasNewValue,
  savingThisSecret,
  secretNameTrimmed,
  canUpdateIntegrations,
  settingsMutationBusy,
  setSecretDrafts,
  setEditingFieldKey,
  exitEdit,
  saveSecret,
}: SecretEditingControlsProps) {
  return (
    <>
      <Input
        id={inputId}
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
        className="min-w-0 flex-1 sm:max-w-xl"
      />
      <div className="flex shrink-0 items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Cancel editing"
          disabled={!canUpdateIntegrations || settingsMutationBusy || savingThisSecret}
          onClick={exitEdit}
        >
          <X className="size-4" aria-hidden />
        </Button>
        <LoadingButton
          type="button"
          color="blue"
          size="sm"
          className="shrink-0"
          disabled={!canUpdateIntegrations || !secretHasNewValue || settingsMutationBusy || !secretNameTrimmed}
          loading={savingThisSecret}
          loadingText="Updating…"
          onClick={async () => {
            if (!secretNameTrimmed) return;
            try {
              await saveSecret(secretNameTrimmed, draft, fieldKey);
              setEditingFieldKey((current) => (current === fieldKey ? null : current));
            } catch {
              // Toast already shown by saveSecret
            }
          }}
        >
          Update
          <Check className="size-4" aria-hidden />
        </LoadingButton>
      </div>
    </>
  );
}

type SecretReadonlyFieldProps = {
  inputId: string;
  title: string;
  isEditable: boolean;
  canUpdateIntegrations: boolean;
  settingsMutationBusy: boolean;
  startEdit: () => void;
};

function SecretReadonlyField({
  inputId,
  title,
  isEditable,
  canUpdateIntegrations,
  settingsMutationBusy,
  startEdit,
}: SecretReadonlyFieldProps) {
  return (
    <div className="flex min-w-0 flex-1 items-center gap-1.5">
      <Input
        id={inputId}
        type="text"
        readOnly
        tabIndex={-1}
        value=""
        placeholder={isEditable ? "Configured" : "Cannot be changed here"}
        aria-readonly="true"
        className={`min-w-0 flex-1 max-w-xl ${READONLY_SECRET_INPUT_CLASS}`}
      />
      {isEditable ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={`Edit ${title}`}
          disabled={!canUpdateIntegrations || settingsMutationBusy}
          onClick={startEdit}
        >
          <Pencil className="size-4" aria-hidden />
        </Button>
      ) : null}
    </div>
  );
}

type IntegrationSecretRowProps = {
  secret: OrganizationsIntegrationSecret;
  index: number;
  secretDrafts: Record<string, string>;
  setSecretDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  editingFieldKey: string | null;
  setEditingFieldKey: Dispatch<SetStateAction<string | null>>;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  settingsMutationBusy: boolean;
  saveSecret: (secretName: string, value: string, draftFieldKey: string) => Promise<void>;
  isSavingSecret: (secretName: string | undefined) => boolean;
};

function IntegrationSecretRow({
  secret,
  index,
  secretDrafts,
  setSecretDrafts,
  editingFieldKey,
  setEditingFieldKey,
  canUpdateIntegrations,
  permissionsLoading,
  settingsMutationBusy,
  saveSecret,
  isSavingSecret,
}: IntegrationSecretRowProps) {
  const fieldKey = secret.name?.trim() || `__secret_${index}`;
  const title = secret.label?.trim() || "Secret";
  const description = secret.description?.trim();
  const isEditable = secret.editable === true;
  const draft = secretDrafts[fieldKey] ?? "";
  const secretHasNewValue = draft.trim() !== "";
  const savingThisSecret = isSavingSecret(secret.name?.trim());
  const isEditingThis = editingFieldKey === fieldKey;
  const inputId = `integration-secret-${fieldKey}`;
  const secretNameTrimmed = secret.name?.trim();

  const exitEdit = () => {
    setEditingFieldKey((current) => (current === fieldKey ? null : current));
    setSecretDrafts((previous) => ({ ...previous, [fieldKey]: "" }));
  };

  const startEdit = () => setEditingFieldKey(fieldKey);

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-gray-200 pb-4 last:border-b-0 last:pb-0 dark:border-gray-800">
      <Label htmlFor={inputId} className="shrink-0 text-gray-800 dark:text-gray-100">
        {title}
      </Label>
      <DescriptionTooltip title={title} description={description} />

      <PermissionTooltip
        allowed={canUpdateIntegrations || permissionsLoading}
        message="You don't have permission to update integrations."
      >
        <div className="flex min-w-[min(100%,240px)] flex-1 flex-wrap items-center gap-2">
          {isEditable && isEditingThis ? (
            <SecretEditingControls
              inputId={inputId}
              fieldKey={fieldKey}
              draft={draft}
              secretHasNewValue={secretHasNewValue}
              savingThisSecret={savingThisSecret}
              secretNameTrimmed={secretNameTrimmed}
              canUpdateIntegrations={canUpdateIntegrations}
              settingsMutationBusy={settingsMutationBusy}
              setSecretDrafts={setSecretDrafts}
              setEditingFieldKey={setEditingFieldKey}
              exitEdit={exitEdit}
              saveSecret={saveSecret}
            />
          ) : (
            <SecretReadonlyField
              inputId={inputId}
              title={title}
              isEditable={isEditable}
              canUpdateIntegrations={canUpdateIntegrations}
              settingsMutationBusy={settingsMutationBusy}
              startEdit={startEdit}
            />
          )}
        </div>
      </PermissionTooltip>
    </div>
  );
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
  const [editingFieldKey, setEditingFieldKey] = useState<string | null>(null);

  return (
    <>
      {integrationSecrets.length > 0 ? (
        <div className="space-y-4 rounded-lg border border-gray-300 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
          {integrationSecrets.map((secret, index) => (
            <IntegrationSecretRow
              key={secret.name?.trim() || `__secret_${index}`}
              secret={secret}
              index={index}
              secretDrafts={secretDrafts}
              setSecretDrafts={setSecretDrafts}
              editingFieldKey={editingFieldKey}
              setEditingFieldKey={setEditingFieldKey}
              canUpdateIntegrations={canUpdateIntegrations}
              permissionsLoading={permissionsLoading}
              settingsMutationBusy={settingsMutationBusy}
              saveSecret={saveSecret}
              isSavingSecret={isSavingSecret}
            />
          ))}
        </div>
      ) : (
        <p className="text-sm text-gray-500 dark:text-gray-400">No secrets stored for this integration.</p>
      )}
    </>
  );
}
