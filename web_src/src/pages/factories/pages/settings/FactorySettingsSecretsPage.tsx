import { useEffect, useMemo, useState } from "react";
import { ChevronLeft, Search } from "lucide-react";
import { useNavigate, useParams } from "react-router";

import type { SuperplaneSecretsSecret } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
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
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateSecret,
  useDeleteSecret,
  useDeleteSecretKey,
  useSecret,
  useSecrets,
  useSetSecretKey,
  useUpdateSecretName,
} from "@/hooks/useSecrets";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { getApiErrorMessage } from "@/lib/errors";
import { useOrganizationSettingsPaths } from "@/lib/organizationSettingsPaths";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";

import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";

const MASKED_VALUE = "••••••••";
const SECRET_DOMAIN = "DOMAIN_TYPE_ORGANIZATION" as const;

type KeyPairDraft = {
  id: string;
  name: string;
  value: string;
};

function formatKeyCount(count: number): string {
  return `${count} key${count === 1 ? "" : "s"}`;
}

function createDraftId(): string {
  return `draft-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
}

function secretName(secret: SuperplaneSecretsSecret): string {
  return secret.metadata?.name || "Unnamed secret";
}

function secretIdOf(secret: SuperplaneSecretsSecret): string {
  return secret.metadata?.id || "";
}

function secretKeyNames(secret: SuperplaneSecretsSecret): string[] {
  return Object.keys(secret.spec?.local?.data || {});
}

function formatUpdatedAt(secret: SuperplaneSecretsSecret): string {
  const createdAt = secret.metadata?.createdAt;
  if (!createdAt) {
    return "Unknown";
  }
  return new Date(createdAt).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function FactorySettingsSecretsPage() {
  const { secretId } = useParams<{ secretId?: string }>();
  if (secretId) {
    return <FactorySettingsSecretDetail secretId={secretId} />;
  }
  return <FactorySettingsSecretsList />;
}

function FactorySettingsSecretsList() {
  const { organizationId } = useFactorySettingsLayout();
  const navigate = useNavigate();
  const settingsPaths = useOrganizationSettingsPaths(organizationId);
  const { data: organization } = useOrganization(organizationId);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canCreate = canAct("secrets", "create");
  const canDelete = canAct("secrets", "delete");
  const { data: secrets = [], isLoading } = useSecrets(organizationId, SECRET_DOMAIN);
  const deleteMutation = useDeleteSecret(organizationId, SECRET_DOMAIN);
  const organizationName = organization?.metadata?.name?.trim() || "this organization";
  const [searchQuery, setSearchQuery] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);

  usePageTitle(["Secrets"]);
  useReportPageReady(!isLoading && !permissionsLoading);

  const filteredSecrets = useMemo(() => {
    const normalized = searchQuery.trim().toLowerCase();
    const sorted = [...secrets].sort((a, b) => secretName(a).localeCompare(secretName(b)));
    if (!normalized) {
      return sorted;
    }
    return sorted.filter((secret) => secretName(secret).toLowerCase().includes(normalized));
  }, [searchQuery, secrets]);

  const renderCreateButton = (testId: string) => (
    <PermissionTooltip allowed={canCreate || permissionsLoading} message="You don't have permission to create secrets.">
      <Button type="button" size="sm" onClick={() => setCreateOpen(true)} disabled={!canCreate} data-testid={testId}>
        Create secret
      </Button>
    </PermissionTooltip>
  );

  if (isLoading) {
    return (
      <FactorySettingsPageFrame title="Secrets" subtitle={`Store credentials for ${organizationName} integrations.`}>
        <FactorySettingsCard>
          <p className="text-[13px] text-muted-foreground">Loading secrets…</p>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>
    );
  }

  return (
    <div data-testid="factory-settings-secrets">
      <FactorySettingsPageFrame
        title="Secrets"
        subtitle={`Store credentials for ${organizationName} integrations. SuperPlane hides values after you save.`}
        actions={renderCreateButton("secrets-create-btn")}
      >
        <FactorySettingsCard data-testid="factory-settings-secrets-list">
          {secrets.length === 0 ? (
            <div className="py-8 text-center" data-testid="factory-settings-secrets-empty">
              <p className="text-[13px] font-medium text-foreground">No secrets yet.</p>
              <p className="mt-1 text-[12px] text-muted-foreground">
                Create a secret to store credentials that integrations can use.
              </p>
              <div className="mt-4 flex justify-center">{renderCreateButton("secrets-create-empty")}</div>
            </div>
          ) : (
            <>
              <div className="relative mb-3">
                <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                  placeholder="Search secrets"
                  className="h-8 pl-8 text-[13px]"
                  data-testid="factory-settings-secrets-search"
                />
              </div>
              {filteredSecrets.length > 0 ? (
                <ul className="divide-y divide-border">
                  {filteredSecrets.map((secret) => {
                    const id = secretIdOf(secret);
                    const name = secretName(secret);
                    return (
                      <li key={id} className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
                        <button
                          type="button"
                          onClick={() => navigate(settingsPaths.secretDetail(id))}
                          className="min-w-0 flex-1 text-left"
                          data-testid="secrets-secret-link"
                        >
                          <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
                          <p className="text-[12px] text-muted-foreground">
                            {formatKeyCount(secretKeyNames(secret).length)} · Updated {formatUpdatedAt(secret)}
                          </p>
                        </button>
                        <PermissionTooltip
                          allowed={canDelete || permissionsLoading}
                          message="You don't have permission to delete secrets."
                        >
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={!canDelete}
                            onClick={() => setDeleteTarget({ id, name })}
                          >
                            Delete
                          </Button>
                        </PermissionTooltip>
                      </li>
                    );
                  })}
                </ul>
              ) : (
                <p className="py-6 text-center text-[13px] text-muted-foreground">No secrets match your search.</p>
              )}
            </>
          )}
        </FactorySettingsCard>
      </FactorySettingsPageFrame>

      <CreateSecretDialog open={createOpen} organizationId={organizationId} onOpenChange={setCreateOpen} />

      <AlertDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete &quot;{deleteTarget?.name}&quot;?</AlertDialogTitle>
            <AlertDialogDescription>
              SuperPlane removes this secret. Integrations that use it fail.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep secret</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={() => {
                if (!deleteTarget) {
                  return;
                }
                void deleteMutation
                  .mutateAsync(deleteTarget.id)
                  .then(() => showSuccessToast("Secret deleted."))
                  .catch((error) => showErrorToast(getApiErrorMessage(error, "Failed to delete secret.")));
                setDeleteTarget(null);
              }}
            >
              Delete secret
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function CreateSecretDialog({
  open,
  organizationId,
  onOpenChange,
}: {
  open: boolean;
  organizationId: string;
  onOpenChange: (open: boolean) => void;
}) {
  const createMutation = useCreateSecret(organizationId, SECRET_DOMAIN);
  const [name, setName] = useState("");
  const [pairs, setPairs] = useState<KeyPairDraft[]>([{ id: createDraftId(), name: "", value: "" }]);
  const [error, setError] = useState("");

  const resetForm = () => {
    setName("");
    setPairs([{ id: createDraftId(), name: "", value: "" }]);
    setError("");
  };

  const handleSubmit = async () => {
    const trimmedName = name.trim();
    const validPairs = pairs
      .map((pair) => ({ name: pair.name.trim(), value: pair.value.trim() }))
      .filter((pair) => pair.name && pair.value);
    if (!trimmedName) {
      setError("Secret name is required.");
      return;
    }
    if (validPairs.length === 0) {
      setError("Add at least one key and value.");
      return;
    }
    const keys = validPairs.map((pair) => pair.name);
    if (new Set(keys).size !== keys.length) {
      setError("Key names must be unique.");
      return;
    }

    try {
      await createMutation.mutateAsync({
        name: trimmedName,
        environmentVariables: validPairs,
      });
      showSuccessToast("Secret created.");
      resetForm();
      onOpenChange(false);
    } catch (createError) {
      showErrorToast(getApiErrorMessage(createError, "Failed to create secret."));
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          resetForm();
        }
        onOpenChange(nextOpen);
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Create secret</DialogTitle>
          <DialogDescription>Store credentials that integrations can use.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="factory-settings-secret-name">Name</Label>
            <Input
              id="factory-settings-secret-name"
              value={name}
              onChange={(event) => {
                setError("");
                setName(event.target.value);
              }}
            />
          </div>
          <div className="space-y-3">
            <p className="text-[13px] font-medium text-foreground">Keys</p>
            {pairs.map((pair, index) => (
              <div key={pair.id} className="grid gap-2 sm:grid-cols-2">
                <Input
                  value={pair.name}
                  placeholder="KEY_NAME"
                  className="font-mono text-[13px]"
                  onChange={(event) =>
                    setPairs((current) =>
                      current.map((item) => (item.id === pair.id ? { ...item, name: event.target.value } : item)),
                    )
                  }
                  data-testid="secrets-create-key"
                />
                <Textarea
                  value={pair.value}
                  placeholder="Value"
                  rows={2}
                  className="font-mono text-[13px]"
                  onChange={(event) =>
                    setPairs((current) =>
                      current.map((item) => (item.id === pair.id ? { ...item, value: event.target.value } : item)),
                    )
                  }
                  data-testid="secrets-create-value"
                />
                {index > 0 ? (
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    className="sm:col-span-2 justify-self-start"
                    onClick={() => setPairs((current) => current.filter((item) => item.id !== pair.id))}
                  >
                    Remove key
                  </Button>
                ) : null}
              </div>
            ))}
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => setPairs((current) => [...current, { id: createDraftId(), name: "", value: "" }])}
            >
              Add key
            </Button>
          </div>
          {error ? <p className="text-[11px] text-destructive">{error}</p> : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <LoadingButton type="button" size="sm" loading={createMutation.isPending} onClick={() => void handleSubmit()}>
            Create secret
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function FactorySettingsSecretDetail({ secretId }: { secretId: string }) {
  const { organizationId } = useFactorySettingsLayout();
  const navigate = useNavigate();
  const settingsPaths = useOrganizationSettingsPaths(organizationId);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("secrets", "update");
  const canDelete = canAct("secrets", "delete");
  const { data: secret, isLoading, error } = useSecret(organizationId, SECRET_DOMAIN, secretId);
  const updateNameMutation = useUpdateSecretName(organizationId, SECRET_DOMAIN, secretId);
  const setKeyMutation = useSetSecretKey(organizationId, SECRET_DOMAIN, secretId);
  const deleteKeyMutation = useDeleteSecretKey(organizationId, SECRET_DOMAIN, secretId);
  const deleteSecretMutation = useDeleteSecret(organizationId, SECRET_DOMAIN);
  const [renameValue, setRenameValue] = useState("");
  const [editingKeyName, setEditingKeyName] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyValue, setNewKeyValue] = useState("");
  const [deleteSecretOpen, setDeleteSecretOpen] = useState(false);
  const [deleteKeyName, setDeleteKeyName] = useState<string | null>(null);

  usePageTitle(["Secrets", secret?.metadata?.name]);
  useReportPageReady(!isLoading && Boolean(secretId), { failed: Boolean(error) });

  useEffect(() => {
    if (secret?.metadata?.name) {
      setRenameValue(secret.metadata.name);
    }
  }, [secret?.metadata?.name]);

  if (isLoading) {
    return (
      <FactorySettingsPageFrame title="Secrets">
        <FactorySettingsCard>
          <p className="text-[13px] text-muted-foreground">Loading secret…</p>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>
    );
  }

  if (!secret) {
    return (
      <FactorySettingsPageFrame title="Secrets">
        <FactorySettingsCard>
          <p className="text-[13px] text-muted-foreground">This secret was not found.</p>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>
    );
  }

  const keys = secretKeyNames(secret);
  const trimmedRename = renameValue.trim();
  const canSaveRename = Boolean(trimmedRename) && trimmedRename !== secretName(secret);
  const canAddKey = Boolean(newKeyName.trim() && newKeyValue.trim());

  return (
    <FactorySettingsPageFrame title="Secrets" subtitle="Edit this secret and its keys.">
      <FactorySettingsCard data-testid="factory-settings-secret-detail">
        <button
          type="button"
          onClick={() => navigate(settingsPaths.secrets)}
          className="mb-4 inline-flex items-center gap-1 text-[12px] text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="size-3.5" aria-hidden />
          All secrets
        </button>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="factory-settings-secret-rename">Secret name</Label>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <Input
                id="factory-settings-secret-rename"
                value={renameValue}
                onChange={(event) => setRenameValue(event.target.value)}
                className="text-[13px]"
                disabled={!canUpdate}
              />
              <PermissionTooltip
                allowed={canUpdate || permissionsLoading}
                message="You don't have permission to update secrets."
              >
                <Button
                  type="button"
                  size="sm"
                  disabled={!canUpdate || !canSaveRename || updateNameMutation.isPending}
                  onClick={() => {
                    void updateNameMutation
                      .mutateAsync(trimmedRename)
                      .then(() => showSuccessToast("Secret name saved."))
                      .catch((renameError) =>
                        showErrorToast(getApiErrorMessage(renameError, "Failed to save the secret name.")),
                      );
                  }}
                >
                  Save
                </Button>
              </PermissionTooltip>
            </div>
            <p className="text-[12px] text-muted-foreground">
              {formatKeyCount(keys.length)} · Updated {formatUpdatedAt(secret)}
            </p>
          </div>

          <div>
            <p className="mb-2 text-[13px] font-medium text-foreground">Keys</p>
            <ul className="divide-y divide-border rounded-md border border-border px-3">
              {keys.map((keyName) => (
                <li key={keyName} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <p className="font-mono text-[13px] text-foreground">{keyName}</p>
                    {editingKeyName === keyName ? (
                      <Textarea
                        value={editValue}
                        onChange={(event) => setEditValue(event.target.value)}
                        rows={2}
                        className="mt-2 font-mono text-[13px]"
                      />
                    ) : (
                      <p className="text-[12px] text-muted-foreground">{MASKED_VALUE}</p>
                    )}
                  </div>
                  <div className="flex gap-2">
                    {editingKeyName === keyName ? (
                      <>
                        <Button
                          type="button"
                          size="sm"
                          disabled={!editValue.trim() || setKeyMutation.isPending}
                          onClick={() => {
                            void setKeyMutation
                              .mutateAsync({ keyName, value: editValue })
                              .then(() => {
                                showSuccessToast("Value updated.");
                                setEditingKeyName(null);
                                setEditValue("");
                              })
                              .catch((updateError) =>
                                showErrorToast(getApiErrorMessage(updateError, "Failed to update the key.")),
                              );
                          }}
                        >
                          Save
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setEditingKeyName(null);
                            setEditValue("");
                          }}
                        >
                          Cancel
                        </Button>
                      </>
                    ) : (
                      <>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled={!canUpdate}
                          onClick={() => {
                            setEditingKeyName(keyName);
                            setEditValue("");
                          }}
                        >
                          Edit
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="ghost"
                          disabled={!canDelete}
                          onClick={() => setDeleteKeyName(keyName)}
                        >
                          Delete
                        </Button>
                      </>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          </div>

          <div className="space-y-3 rounded-md border border-border p-3">
            <p className="text-[13px] font-medium text-foreground">Add key</p>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="factory-settings-add-key-name">Key name</Label>
                <Input
                  id="factory-settings-add-key-name"
                  value={newKeyName}
                  onChange={(event) => setNewKeyName(event.target.value)}
                  placeholder="WEBHOOK_URL"
                  className="font-mono text-[13px]"
                  disabled={!canUpdate}
                />
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="factory-settings-add-key-value">Value</Label>
                <Textarea
                  id="factory-settings-add-key-value"
                  value={newKeyValue}
                  onChange={(event) => setNewKeyValue(event.target.value)}
                  rows={2}
                  className="font-mono text-[13px]"
                  disabled={!canUpdate}
                />
              </div>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!canUpdate || !canAddKey || setKeyMutation.isPending}
              onClick={() => {
                void setKeyMutation
                  .mutateAsync({ keyName: newKeyName.trim(), value: newKeyValue })
                  .then(() => {
                    showSuccessToast("Key added.");
                    setNewKeyName("");
                    setNewKeyValue("");
                  })
                  .catch((addError) => showErrorToast(getApiErrorMessage(addError, "Failed to add the key.")));
              }}
            >
              Add key
            </Button>
          </div>

          <div className="border-t border-border pt-4">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0 space-y-0.5">
                <p className="text-[13px] font-medium text-foreground">Delete secret</p>
                <p className="text-[12px] text-muted-foreground">
                  SuperPlane removes this secret and all keys. Integrations that use it fail.
                </p>
              </div>
              <PermissionTooltip
                allowed={canDelete || permissionsLoading}
                message="You don't have permission to delete secrets."
              >
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  disabled={!canDelete}
                  onClick={() => setDeleteSecretOpen(true)}
                >
                  Delete secret
                </Button>
              </PermissionTooltip>
            </div>
          </div>
        </div>
      </FactorySettingsCard>

      <AlertDialog open={deleteSecretOpen} onOpenChange={setDeleteSecretOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete &quot;{secretName(secret)}&quot;?</AlertDialogTitle>
            <AlertDialogDescription>
              SuperPlane removes this secret. Integrations that use it fail.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep secret</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={() => {
                void deleteSecretMutation
                  .mutateAsync(secretId)
                  .then(() => {
                    showSuccessToast("Secret deleted.");
                    navigate(settingsPaths.secrets);
                  })
                  .catch((deleteError) => showErrorToast(getApiErrorMessage(deleteError, "Failed to delete secret.")));
              }}
            >
              Delete secret
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={Boolean(deleteKeyName)}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteKeyName(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete key &quot;{deleteKeyName}&quot; from &quot;{secretName(secret)}&quot;?
            </AlertDialogTitle>
            <AlertDialogDescription>SuperPlane removes this key. Integrations that use it fail.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep key</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={() => {
                if (!deleteKeyName) {
                  return;
                }
                void deleteKeyMutation
                  .mutateAsync(deleteKeyName)
                  .then(() => showSuccessToast("Key deleted."))
                  .catch((deleteError) => showErrorToast(getApiErrorMessage(deleteError, "Failed to delete the key.")));
                setDeleteKeyName(null);
              }}
            >
              Delete key
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FactorySettingsPageFrame>
  );
}
