import { useEffect, useMemo, useState } from "react";
import { ChevronLeft, Search } from "lucide-react";
import { useNavigate, useParams } from "react-router";

import type { ApiKeysApiKey } from "@/api-client/types.gen";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useAPIKey,
  useAPIKeys,
  useCreateAPIKey,
  useDeleteAPIKey,
  useRegenerateAPIKeyToken,
  useUpdateAPIKey,
} from "@/hooks/useApiKeys";
import { useCanvases } from "@/hooks/useCanvasData";
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
import { CopyButton } from "@/ui/CopyButton";

import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";

type AccessMode = "organization" | "canvas";
type ExpirationPreset = "never" | "30d" | "90d" | "custom";

type CreateFormState = {
  name: string;
  description: string;
  role: string;
  accessMode: AccessMode;
  canvasIds: string[];
  expirationPreset: ExpirationPreset;
  customExpiresAt: string;
};

const EMPTY_CREATE_FORM: CreateFormState = {
  name: "",
  description: "",
  role: "org_viewer",
  accessMode: "organization",
  canvasIds: [],
  expirationPreset: "never",
  customExpiresAt: "",
};

function apiKeyName(apiKey: ApiKeysApiKey): string {
  return apiKey.name || "Unnamed";
}

function apiKeyIdOf(apiKey: ApiKeysApiKey): string {
  return apiKey.id || "";
}

function addDays(from: Date, days: number): Date {
  const next = new Date(from);
  next.setDate(next.getDate() + days);
  return next;
}

function toLocalDateTimeValue(date: Date): string {
  const offset = date.getTimezoneOffset();
  const local = new Date(date.getTime() - offset * 60_000);
  return local.toISOString().slice(0, 16);
}

function computeExpiresAt(preset: ExpirationPreset, customExpiresAt: string): string | undefined {
  if (preset === "never") {
    return undefined;
  }
  if (preset === "30d") {
    return addDays(new Date(), 30).toISOString();
  }
  if (preset === "90d") {
    return addDays(new Date(), 90).toISOString();
  }
  if (!customExpiresAt) {
    return undefined;
  }
  return new Date(customExpiresAt).toISOString();
}

function isKeyExpired(apiKey: ApiKeysApiKey, now = Date.now()): boolean {
  if (!apiKey.expiresAt) {
    return false;
  }
  return new Date(apiKey.expiresAt).getTime() <= now;
}

function formatShortDate(value?: string): string {
  if (!value) {
    return "Never";
  }
  return new Date(value).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function formatAccessLabel(apiKey: ApiKeysApiKey, canvasNamesById: Map<string, string>): string {
  const canvasIds = apiKey.canvasIds ?? [];
  if (canvasIds.length === 0) {
    return "Organization-wide";
  }
  if (canvasIds.length === 1) {
    return canvasNamesById.get(canvasIds[0] ?? "") ?? "1 app";
  }
  return `${canvasIds.length} apps`;
}

function formatExpirationLabel(apiKey: ApiKeysApiKey): string {
  if (!apiKey.expiresAt) {
    return "Never expires";
  }
  if (isKeyExpired(apiKey)) {
    return `Expired ${formatShortDate(apiKey.expiresAt)}`;
  }
  return `Expires ${formatShortDate(apiKey.expiresAt)}`;
}

function formatMetadataLine(apiKey: ApiKeysApiKey, canvasNamesById: Map<string, string>): string {
  return [formatAccessLabel(apiKey, canvasNamesById), formatExpirationLabel(apiKey)].join(" · ");
}

export function FactorySettingsApiKeysPage() {
  const { id: apiKeyId } = useParams<{ id?: string }>();
  if (apiKeyId) {
    return <FactorySettingsApiKeyDetail apiKeyId={apiKeyId} />;
  }
  return <FactorySettingsApiKeysList />;
}

function FactorySettingsApiKeysList() {
  const { organizationId } = useFactorySettingsLayout();
  const navigate = useNavigate();
  const settingsPaths = useOrganizationSettingsPaths(organizationId);
  const { data: organization } = useOrganization(organizationId);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canCreate = canAct("api_keys", "create");
  const canDelete = canAct("api_keys", "delete");
  const { data: apiKeys = [], isLoading } = useAPIKeys(organizationId);
  const { data: canvases = [] } = useCanvases(organizationId);
  const deleteMutation = useDeleteAPIKey(organizationId);
  const organizationName = organization?.metadata?.name?.trim() || "this organization";
  const canvasNamesById = useMemo(
    () => new Map(canvases.map((canvas) => [canvas.id || "", canvas.name || "Unnamed"])),
    [canvases],
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<ApiKeysApiKey | null>(null);

  usePageTitle(["API keys"]);
  useReportPageReady(!isLoading && !permissionsLoading);

  const filteredApiKeys = useMemo(() => {
    const normalized = searchQuery.trim().toLowerCase();
    const sorted = [...apiKeys].sort((a, b) => apiKeyName(a).localeCompare(apiKeyName(b)));
    if (!normalized) {
      return sorted;
    }
    return sorted.filter((apiKey) => apiKeyName(apiKey).toLowerCase().includes(normalized));
  }, [searchQuery, apiKeys]);

  const renderCreateButton = (testId: string) => (
    <PermissionTooltip
      allowed={canCreate || permissionsLoading}
      message="You don't have permission to create API keys."
    >
      <Button type="button" size="sm" onClick={() => setCreateOpen(true)} disabled={!canCreate} data-testid={testId}>
        Create API key
      </Button>
    </PermissionTooltip>
  );

  if (isLoading) {
    return (
      <FactorySettingsPageFrame title="API keys" subtitle={`Organization API keys for ${organizationName}.`}>
        <FactorySettingsCard>
          <p className="text-[13px] text-muted-foreground">Loading API keys…</p>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>
    );
  }

  return (
    <div data-testid="factory-settings-api-keys">
      <FactorySettingsPageFrame
        title="API keys"
        subtitle={`Organization API keys for ${organizationName}. These are not personal access tokens.`}
        actions={renderCreateButton("api-key-create-btn")}
      >
        <FactorySettingsCard data-testid="factory-settings-api-keys-list">
          {apiKeys.length === 0 ? (
            <div className="py-8 text-center" data-testid="factory-settings-api-keys-empty">
              <p className="text-[13px] font-medium text-foreground">No API keys yet.</p>
              <p className="mt-1 text-[12px] text-muted-foreground">
                Create a key to authenticate scripts for this organization.
              </p>
              <div className="mt-4 flex justify-center">{renderCreateButton("api-key-create-empty")}</div>
            </div>
          ) : (
            <>
              <div className="relative mb-3">
                <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                  placeholder="Search API keys"
                  className="h-8 pl-8 text-[13px]"
                  data-testid="factory-settings-api-keys-search"
                />
              </div>
              {filteredApiKeys.length > 0 ? (
                <ul className="divide-y divide-border">
                  {filteredApiKeys.map((apiKey) => {
                    const id = apiKeyIdOf(apiKey);
                    const name = apiKeyName(apiKey);
                    return (
                      <li key={id} className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
                        <button
                          type="button"
                          onClick={() => navigate(settingsPaths.apiKeyDetail(id))}
                          className="min-w-0 flex-1 text-left"
                          data-testid="api-key-link"
                        >
                          <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
                          <p className="text-[12px] text-muted-foreground">
                            {formatMetadataLine(apiKey, canvasNamesById)}
                          </p>
                        </button>
                        <PermissionTooltip
                          allowed={canDelete || permissionsLoading}
                          message="You don't have permission to delete API keys."
                        >
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={!canDelete || deleteMutation.isPending}
                            onClick={() => setRevokeTarget(apiKey)}
                            data-testid="api-key-delete-btn"
                          >
                            Revoke
                          </Button>
                        </PermissionTooltip>
                      </li>
                    );
                  })}
                </ul>
              ) : (
                <p className="py-6 text-center text-[13px] text-muted-foreground">No API keys match your search.</p>
              )}
            </>
          )}
        </FactorySettingsCard>
      </FactorySettingsPageFrame>

      <CreateApiKeyDialog
        open={createOpen}
        organizationId={organizationId}
        canvases={canvases}
        onOpenChange={setCreateOpen}
        onCreated={(token) => {
          if (token) {
            setRevealedToken(token);
          }
        }}
      />

      <RevealTokenDialog
        token={revealedToken}
        title="API key created"
        onOpenChange={(open) => {
          if (!open) {
            setRevealedToken(null);
          }
        }}
      />

      <RevokeApiKeyDialog
        apiKey={revokeTarget}
        onOpenChange={(open) => {
          if (!open) {
            setRevokeTarget(null);
          }
        }}
        onConfirm={(apiKeyId) => {
          void deleteMutation
            .mutateAsync(apiKeyId)
            .then(() => showSuccessToast("API key revoked."))
            .catch((error) => showErrorToast(`Failed to delete: ${getApiErrorMessage(error)}`));
        }}
      />
    </div>
  );
}

function CreateApiKeyDialog({
  open,
  organizationId,
  canvases,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  organizationId: string;
  canvases: { id?: string; name?: string }[];
  onOpenChange: (open: boolean) => void;
  onCreated: (token?: string) => void;
}) {
  const createMutation = useCreateAPIKey(organizationId);
  const [form, setForm] = useState<CreateFormState>(EMPTY_CREATE_FORM);
  const [nameError, setNameError] = useState("");
  const [canvasError, setCanvasError] = useState("");

  const resetForm = () => {
    setForm(EMPTY_CREATE_FORM);
    setNameError("");
    setCanvasError("");
  };

  const handleCreate = async () => {
    const trimmedName = form.name.trim();
    if (!trimmedName) {
      setNameError("Name is required.");
      return;
    }
    if (form.accessMode === "canvas" && form.canvasIds.length === 0) {
      setCanvasError("Select at least one app.");
      return;
    }

    try {
      const result = await createMutation.mutateAsync({
        name: trimmedName,
        description: form.description.trim(),
        role: form.role,
        expiresAt: computeExpiresAt(form.expirationPreset, form.customExpiresAt),
        canvasIds: form.accessMode === "canvas" ? form.canvasIds : [],
      });
      resetForm();
      onOpenChange(false);
      onCreated(result.data?.token);
      showSuccessToast("API key created.");
    } catch (error) {
      showErrorToast(`Failed to create API key: ${getApiErrorMessage(error)}`);
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
      <DialogContent className="max-w-lg" data-testid="api-key-create-form">
        <DialogHeader>
          <DialogTitle>Create API key</DialogTitle>
          <DialogDescription>Create an organization API key for scripts and automation.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="factory-settings-api-key-name">Name</Label>
            <Input
              id="factory-settings-api-key-name"
              value={form.name}
              placeholder="ci-deploy-bot"
              onChange={(event) => {
                setNameError("");
                setForm((current) => ({ ...current, name: event.target.value }));
              }}
              data-testid="api-key-create-name"
            />
            {nameError ? <p className="text-[11px] text-destructive">{nameError}</p> : null}
          </div>
          <div className="space-y-2">
            <Label htmlFor="factory-settings-api-key-description">Description</Label>
            <Textarea
              id="factory-settings-api-key-description"
              value={form.description}
              rows={3}
              placeholder="What will this key access?"
              onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
              data-testid="api-key-create-description"
            />
            <p className="text-[12px] text-muted-foreground">Optional. Helps teammates identify this key later.</p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="factory-settings-api-key-role">Role</Label>
            <Select value={form.role} onValueChange={(value) => setForm((current) => ({ ...current, role: value }))}>
              <SelectTrigger id="factory-settings-api-key-role" className="w-full" data-testid="api-key-create-role">
                <SelectValue placeholder="Select a role" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="org_viewer">Viewer</SelectItem>
                <SelectItem value="org_admin">Admin</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-[12px] text-muted-foreground">Sets what this key can do within its access scope.</p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="factory-settings-api-key-access">Access</Label>
            <Select
              value={form.accessMode}
              onValueChange={(value) =>
                setForm((current) => ({
                  ...current,
                  accessMode: value as AccessMode,
                  canvasIds: value === "organization" ? [] : current.canvasIds,
                }))
              }
            >
              <SelectTrigger
                id="factory-settings-api-key-access"
                className="w-full"
                data-testid="api-key-create-access-mode"
              >
                <SelectValue placeholder="Select access" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="organization">Organization-wide</SelectItem>
                <SelectItem value="canvas">Selected apps</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {form.accessMode === "canvas" ? (
            <div className="space-y-2">
              <Label>Apps</Label>
              <div className="max-h-44 overflow-y-auto rounded-md border border-border">
                {canvases.map((canvas) => {
                  const canvasId = canvas.id || "";
                  return (
                    <label
                      key={canvasId}
                      className="flex items-center gap-2 border-b border-border px-3 py-2 text-[13px] last:border-b-0"
                    >
                      <Checkbox
                        checked={form.canvasIds.includes(canvasId)}
                        onChange={() => {
                          setCanvasError("");
                          setForm((current) => ({
                            ...current,
                            canvasIds: current.canvasIds.includes(canvasId)
                              ? current.canvasIds.filter((id) => id !== canvasId)
                              : [...current.canvasIds, canvasId],
                          }));
                        }}
                        data-testid="api-key-create-canvas"
                      />
                      <span>{canvas.name || "Unnamed"}</span>
                    </label>
                  );
                })}
              </div>
              {canvasError ? <p className="text-[11px] text-destructive">{canvasError}</p> : null}
            </div>
          ) : null}
          <div className="space-y-2">
            <Label htmlFor="factory-settings-api-key-expiration">Expiration</Label>
            <Select
              value={form.expirationPreset}
              onValueChange={(value) =>
                setForm((current) => ({
                  ...current,
                  expirationPreset: value as ExpirationPreset,
                  customExpiresAt:
                    value === "custom" && !current.customExpiresAt
                      ? toLocalDateTimeValue(addDays(new Date(), 30))
                      : current.customExpiresAt,
                }))
              }
            >
              <SelectTrigger id="factory-settings-api-key-expiration" className="w-full">
                <SelectValue placeholder="Select expiration" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="never">Never</SelectItem>
                <SelectItem value="30d">30 days</SelectItem>
                <SelectItem value="90d">90 days</SelectItem>
                <SelectItem value="custom">Custom date</SelectItem>
              </SelectContent>
            </Select>
            {form.expirationPreset === "custom" ? (
              <Input
                type="datetime-local"
                value={form.customExpiresAt}
                onChange={(event) => setForm((current) => ({ ...current, customExpiresAt: event.target.value }))}
                data-testid="api-key-create-expires-at"
              />
            ) : null}
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <LoadingButton
            type="button"
            size="sm"
            loading={createMutation.isPending}
            onClick={() => void handleCreate()}
            data-testid="api-key-create-submit"
          >
            Create API key
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RevealTokenDialog({
  token,
  title,
  onOpenChange,
}: {
  token: string | null;
  title: string;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog
      open={Boolean(token)}
      onOpenChange={(open) => {
        if (!open) {
          onOpenChange(false);
        }
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>Copy this token now. You cannot see it again.</DialogDescription>
        </DialogHeader>
        {token ? (
          <div className="flex items-center gap-2">
            <Input readOnly value={token} className="font-mono text-[12px]" data-testid="api-key-token-display" />
            <CopyButton variant="button" text={token} copiedLabel="Copied" data-testid="api-key-token-copy">
              Copy
            </CopyButton>
          </div>
        ) : null}
        <DialogFooter>
          <Button type="button" size="sm" onClick={() => onOpenChange(false)} data-testid="api-key-token-done">
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RevokeApiKeyDialog({
  apiKey,
  onOpenChange,
  onConfirm,
}: {
  apiKey: ApiKeysApiKey | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: (apiKeyId: string) => void;
}) {
  return (
    <AlertDialog
      open={Boolean(apiKey)}
      onOpenChange={(open) => {
        if (!open) {
          onOpenChange(false);
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke &quot;{apiKey?.name}&quot;?</AlertDialogTitle>
          <AlertDialogDescription>SuperPlane deletes this key. Scripts that use it fail.</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Keep API key</AlertDialogCancel>
          <AlertDialogAction
            className="bg-destructive text-white hover:bg-destructive/90"
            onClick={() => {
              const id = apiKey?.id;
              if (!id) {
                return;
              }
              onConfirm(id);
              onOpenChange(false);
            }}
          >
            Revoke API key
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function FactorySettingsApiKeyDetail({ apiKeyId }: { apiKeyId: string }) {
  const { organizationId } = useFactorySettingsLayout();
  const navigate = useNavigate();
  const settingsPaths = useOrganizationSettingsPaths(organizationId);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("api_keys", "update");
  const canDelete = canAct("api_keys", "delete");
  const { data: apiKey, isLoading, error } = useAPIKey(organizationId, apiKeyId);
  const { data: canvases = [] } = useCanvases(organizationId);
  const updateMutation = useUpdateAPIKey(organizationId);
  const deleteMutation = useDeleteAPIKey(organizationId);
  const regenerateMutation = useRegenerateAPIKeyToken(organizationId);
  const canvasNamesById = useMemo(
    () => new Map(canvases.map((canvas) => [canvas.id || "", canvas.name || "Unnamed"])),
    [canvases],
  );
  const [renameValue, setRenameValue] = useState("");
  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [revokeOpen, setRevokeOpen] = useState(false);

  usePageTitle(["API keys", apiKey?.name]);
  useReportPageReady(!isLoading && Boolean(apiKeyId), { failed: Boolean(error) });

  useEffect(() => {
    if (apiKey?.name) {
      setRenameValue(apiKey.name);
    }
  }, [apiKey?.name]);

  if (isLoading) {
    return (
      <FactorySettingsPageFrame title="API keys">
        <FactorySettingsCard>
          <p className="text-[13px] text-muted-foreground">Loading API key…</p>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>
    );
  }

  if (!apiKey) {
    return (
      <FactorySettingsPageFrame title="API keys">
        <FactorySettingsCard>
          <p className="text-[13px] text-muted-foreground">This API key was not found.</p>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>
    );
  }

  const trimmedRename = renameValue.trim();
  const canSaveRename = Boolean(trimmedRename) && trimmedRename !== apiKeyName(apiKey);
  const detailRows = [
    { label: "Description", value: apiKey.description?.trim() || "—" },
    { label: "Access", value: formatAccessLabel(apiKey, canvasNamesById) },
    { label: "Status", value: isKeyExpired(apiKey) ? "Expired" : "Active" },
    { label: "Created by", value: apiKey.createdByName?.trim() || "—" },
    { label: "Expiration", value: formatExpirationLabel(apiKey) },
  ];

  return (
    <FactorySettingsPageFrame title="API keys" subtitle="Edit this API key.">
      <FactorySettingsCard data-testid="factory-settings-api-key-detail">
        <button
          type="button"
          onClick={() => navigate(settingsPaths.apiKeys)}
          className="mb-4 inline-flex items-center gap-1 text-[12px] text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="size-3.5" aria-hidden />
          All API keys
        </button>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="factory-settings-api-key-rename">API key name</Label>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <Input
                id="factory-settings-api-key-rename"
                value={renameValue}
                onChange={(event) => setRenameValue(event.target.value)}
                className="text-[13px]"
                disabled={!canUpdate}
              />
              <PermissionTooltip
                allowed={canUpdate || permissionsLoading}
                message="You don't have permission to update API keys."
              >
                <Button
                  type="button"
                  size="sm"
                  disabled={!canUpdate || !canSaveRename || updateMutation.isPending}
                  onClick={() => {
                    void updateMutation
                      .mutateAsync({
                        id: apiKeyId,
                        name: trimmedRename,
                        description: apiKey.description ?? "",
                        clearExpiresAt: false,
                      })
                      .then(() => showSuccessToast("API key name saved."))
                      .catch((renameError) =>
                        showErrorToast(getApiErrorMessage(renameError, "Failed to save the API key name.")),
                      );
                  }}
                >
                  Save
                </Button>
              </PermissionTooltip>
            </div>
            <p className="text-[12px] text-muted-foreground">{formatMetadataLine(apiKey, canvasNamesById)}</p>
          </div>

          <div>
            <p className="mb-2 text-[13px] font-medium text-foreground">Details</p>
            <ul className="divide-y divide-border rounded-md border border-border px-3">
              {detailRows.map((row) => (
                <li key={row.label} className="flex flex-col gap-0.5 py-3">
                  <p className="text-[13px] font-medium text-foreground">{row.label}</p>
                  <p className="text-[12px] text-muted-foreground">{row.value}</p>
                </li>
              ))}
            </ul>
          </div>

          <div className="space-y-3 rounded-md border border-border p-3">
            <p className="text-[13px] font-medium text-foreground">Token</p>
            <p className="text-[12px] text-muted-foreground">
              Copy the token when SuperPlane shows it. You cannot see it again.
            </p>
            <PermissionTooltip
              allowed={canUpdate || permissionsLoading}
              message="You don't have permission to update API keys."
            >
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={!canUpdate || regenerateMutation.isPending}
                onClick={() => {
                  void regenerateMutation
                    .mutateAsync(apiKeyId)
                    .then((result) => {
                      const token = result.data?.token;
                      if (token) {
                        setRevealedToken(token);
                      }
                      showSuccessToast("Token regenerated.");
                    })
                    .catch((regenerateError) =>
                      showErrorToast(`Failed to regenerate token: ${getApiErrorMessage(regenerateError)}`),
                    );
                }}
              >
                Regenerate token
              </Button>
            </PermissionTooltip>
          </div>

          <div className="border-t border-border pt-4">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0 space-y-0.5">
                <p className="text-[13px] font-medium text-foreground">Revoke API key</p>
                <p className="text-[12px] text-muted-foreground">
                  SuperPlane deletes this key. Scripts that use it fail.
                </p>
              </div>
              <PermissionTooltip
                allowed={canDelete || permissionsLoading}
                message="You don't have permission to delete API keys."
              >
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  disabled={!canDelete}
                  onClick={() => setRevokeOpen(true)}
                >
                  Revoke API key
                </Button>
              </PermissionTooltip>
            </div>
          </div>
        </div>
      </FactorySettingsCard>

      <RevealTokenDialog
        token={revealedToken}
        title="Token regenerated"
        onOpenChange={(open) => {
          if (!open) {
            setRevealedToken(null);
          }
        }}
      />

      <RevokeApiKeyDialog
        apiKey={revokeOpen ? apiKey : null}
        onOpenChange={(open) => setRevokeOpen(open)}
        onConfirm={(id) => {
          void deleteMutation
            .mutateAsync(id)
            .then(() => {
              showSuccessToast("API key revoked.");
              navigate(settingsPaths.apiKeys);
            })
            .catch((deleteError) => showErrorToast(`Failed to delete: ${getApiErrorMessage(deleteError)}`));
        }}
      />
    </FactorySettingsPageFrame>
  );
}
