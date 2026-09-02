import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useDeleteFactory, useUpdateFactory } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { factoryListPath, factorySettingsGeneralPathAfterKeyChange } from "../../lib/factoryPagePaths";
import { clearLastVisitedFactory } from "../../lib/lastVisitedFactory";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import {
  WORKSPACE_KEY_MAX_LENGTH,
  WORKSPACE_KEY_MIN_LENGTH,
  isValidWorkspaceKey,
  normalizeWorkspaceKey,
} from "../../lib/workspaceKey";

const MAX_NAME_LENGTH = 128;
const MAX_DESCRIPTION_LENGTH = 500;

export function FactorySettingsGeneralPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { account } = useAccount();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const deleteFactory = useDeleteFactory(organizationId);

  usePageTitle(["General", "Settings", factory.name ?? "Workspace"]);

  const [name, setName] = useState(factory.name ?? "");
  const [description, setDescription] = useState(factory.description ?? "");
  const [key, setKey] = useState(factory.key ?? "");
  const [nameError, setNameError] = useState("");
  const [keyError, setKeyError] = useState("");
  const [deleteOpen, setDeleteOpen] = useState(false);

  useEffect(() => {
    setName(factory.name ?? "");
    setDescription(factory.description ?? "");
    setKey(factory.key ?? "");
    setNameError("");
    setKeyError("");
  }, [factory.name, factory.description, factory.key]);

  const canUpdate = canAct("factories", "update");
  const canDelete = canAct("factories", "delete");

  const isDirty =
    name.trim() !== (factory.name ?? "") ||
    description.trim() !== (factory.description ?? "") ||
    key !== (factory.key ?? "");

  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError("Name is required");
      return;
    }
    if (!isValidWorkspaceKey(key)) {
      setKeyError(`Use ${WORKSPACE_KEY_MIN_LENGTH} to ${WORKSPACE_KEY_MAX_LENGTH} uppercase letters.`);
      return;
    }
    try {
      const nextSettingsPath = factorySettingsGeneralPathAfterKeyChange(organizationId, factory.key ?? "", key);
      await updateFactory.mutateAsync({
        name: trimmedName,
        description: description.trim(),
        key: key !== (factory.key ?? "") ? key : undefined,
      });
      showSuccessToast("Workspace updated.");
      if (nextSettingsPath) {
        navigate(nextSettingsPath, { replace: true });
      }
    } catch (error) {
      const message = getApiErrorMessage(error, "Failed to update workspace");
      if (message.toLowerCase().includes("workspace key")) {
        setKeyError(message);
        return;
      }
      showErrorToast(message);
    }
  };

  const handleDelete = async () => {
    try {
      await deleteFactory.mutateAsync(factoryId);
      clearLastVisitedFactory(account?.id ?? "", organizationId, factoryId);
      showSuccessToast("Workspace deleted.");
      navigate(factoryListPath(organizationId));
    } catch {
      showErrorToast("Failed to delete workspace.");
      throw new Error("Failed to delete workspace");
    }
  };

  return (
    <>
      <FactorySettingsPageFrame title="General" subtitle="Name, key, and description for this workspace.">
        <WorkspaceDetailsSection
          name={name}
          description={description}
          factoryKey={key}
          nameError={nameError}
          keyError={keyError}
          canUpdate={canUpdate}
          permissionsLoading={permissionsLoading}
          isSaving={updateFactory.isPending}
          isDirty={isDirty}
          onNameChange={(next) => {
            setName(next);
            if (nameError) setNameError("");
          }}
          onDescriptionChange={setDescription}
          onKeyChange={(next) => {
            setKey(normalizeWorkspaceKey(next));
            if (keyError) setKeyError("");
          }}
          onSave={handleSave}
        />

        <DangerZoneSection
          canDelete={canDelete}
          permissionsLoading={permissionsLoading}
          onOpenDelete={() => setDeleteOpen(true)}
        />
      </FactorySettingsPageFrame>

      <FactoryDeleteDialog
        open={deleteOpen}
        factoryName={factory.name ?? ""}
        canDelete={canDelete}
        isDeleting={deleteFactory.isPending}
        onClose={() => setDeleteOpen(false)}
        onConfirm={handleDelete}
      />
    </>
  );
}

interface WorkspaceDetailsSectionProps {
  name: string;
  description: string;
  factoryKey: string;
  nameError: string;
  keyError: string;
  canUpdate: boolean;
  permissionsLoading: boolean;
  isSaving: boolean;
  isDirty: boolean;
  onNameChange: (next: string) => void;
  onDescriptionChange: (next: string) => void;
  onKeyChange: (next: string) => void;
  onSave: () => Promise<void> | void;
}

function WorkspaceDetailsSection({
  name,
  description,
  factoryKey,
  nameError,
  keyError,
  canUpdate,
  permissionsLoading,
  isSaving,
  isDirty,
  onNameChange,
  onDescriptionChange,
  onKeyChange,
  onSave,
}: WorkspaceDetailsSectionProps) {
  return (
    <section
      className="space-y-4 scroll-mt-8"
      data-testid="factory-settings-general-form"
      id="factory-settings-general-form"
    >
      <div className="space-y-2">
        <Label htmlFor="factory-settings-name">Name</Label>
        <Input
          id="factory-settings-name"
          data-testid="factory-settings-name"
          value={name}
          onChange={(event) => {
            if (event.target.value.length <= MAX_NAME_LENGTH) {
              onNameChange(event.target.value);
            }
          }}
          maxLength={MAX_NAME_LENGTH}
          disabled={!canUpdate}
        />
        {nameError ? <p className="text-[11px] text-destructive">{nameError}</p> : null}
      </div>
      <div className="space-y-2 scroll-mt-8" id="factory-settings-key">
        <Label htmlFor="factory-settings-key-input">Workspace key</Label>
        <Input
          id="factory-settings-key-input"
          data-testid="factory-settings-key"
          value={factoryKey}
          onChange={(event) => onKeyChange(event.target.value)}
          maxLength={WORKSPACE_KEY_MAX_LENGTH}
          disabled={!canUpdate}
          className="max-w-xs uppercase tracking-wider"
          autoComplete="off"
        />
        <p className="text-[11px] text-muted-foreground">
          Changing the key updates every task identifier for this workspace. IDs already shared elsewhere will no longer
          resolve.
        </p>
        {keyError ? <p className="text-[11px] text-destructive">{keyError}</p> : null}
      </div>
      <div className="space-y-2">
        <Label htmlFor="factory-settings-description">Description</Label>
        <Textarea
          id="factory-settings-description"
          data-testid="factory-settings-description"
          value={description}
          onChange={(event) => {
            if (event.target.value.length <= MAX_DESCRIPTION_LENGTH) {
              onDescriptionChange(event.target.value);
            }
          }}
          rows={3}
          disabled={!canUpdate}
        />
      </div>
      <PermissionTooltip
        allowed={canUpdate || permissionsLoading}
        message="You don't have permission to update workspaces."
      >
        <LoadingButton
          disabled={!canUpdate || !name.trim() || !isDirty}
          loading={isSaving}
          loadingText="Saving..."
          onClick={() => void onSave()}
          data-testid="factory-settings-save"
        >
          Save
        </LoadingButton>
      </PermissionTooltip>
    </section>
  );
}

interface DangerZoneSectionProps {
  canDelete: boolean;
  permissionsLoading: boolean;
  onOpenDelete: () => void;
}

function DangerZoneSection({ canDelete, permissionsLoading, onOpenDelete }: DangerZoneSectionProps) {
  return (
    <FactorySettingsCard
      title="Danger zone"
      titleClassName="text-destructive"
      className="border-destructive/40"
      data-testid="factory-settings-danger-zone"
    >
      <p className="text-[13px] text-muted-foreground">
        Deleting a workspace permanently removes its tasks, lines, and apps.
      </p>
      <div className="mt-4">
        <PermissionTooltip
          allowed={canDelete || permissionsLoading}
          message="You don't have permission to delete workspaces."
        >
          <Button
            type="button"
            variant="destructive"
            disabled={!canDelete}
            onClick={onOpenDelete}
            data-testid="factory-settings-delete-button"
          >
            <Trash2 className="h-4 w-4" aria-hidden />
            Delete workspace
          </Button>
        </PermissionTooltip>
      </div>
    </FactorySettingsCard>
  );
}
