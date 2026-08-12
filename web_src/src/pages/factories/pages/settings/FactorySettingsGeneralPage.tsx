import { Heading } from "@/components/Heading/heading";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { usePermissions } from "@/contexts/usePermissions";
import { useDeleteFactory, useUpdateFactory } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { factoryListPath } from "../../lib/factoryPagePaths";
import {
  factoryCardClassName,
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "../factoryPageLayoutStyles";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { cn } from "@/lib/utils";

const MAX_NAME_LENGTH = 128;
const MAX_DESCRIPTION_LENGTH = 500;

export function FactorySettingsGeneralPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const deleteFactory = useDeleteFactory(organizationId);

  const [name, setName] = useState(factory.name ?? "");
  const [description, setDescription] = useState(factory.description ?? "");
  const [nameError, setNameError] = useState("");
  const [deleteOpen, setDeleteOpen] = useState(false);

  useEffect(() => {
    setName(factory.name ?? "");
    setDescription(factory.description ?? "");
    setNameError("");
  }, [factory.name, factory.description]);

  const canUpdate = canAct("factories", "update");
  const canDelete = canAct("factories", "delete");

  const isDirty = name.trim() !== (factory.name ?? "") || description.trim() !== (factory.description ?? "");

  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError("Name is required");
      return;
    }
    try {
      await updateFactory.mutateAsync({ name: trimmedName, description: description.trim() });
      showSuccessToast("Workspace updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update workspace"));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteFactory.mutateAsync(factoryId);
      showSuccessToast("Workspace deleted.");
      navigate(factoryListPath(organizationId));
    } catch {
      showErrorToast("Failed to delete workspace.");
      throw new Error("Failed to delete workspace");
    }
  };

  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div>
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            General
          </Heading>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>
            General workspace preferences. Content for this page comes next.
          </p>
        </div>
      </header>

      <div className={factoryContentBodyClassName}>
        <div className="max-w-2xl space-y-6">
          <WorkspaceDetailsSection
            name={name}
            description={description}
            nameError={nameError}
            canUpdate={canUpdate}
            permissionsLoading={permissionsLoading}
            isSaving={updateFactory.isPending}
            isDirty={isDirty}
            onNameChange={(next) => {
              setName(next);
              if (nameError) setNameError("");
            }}
            onDescriptionChange={setDescription}
            onSave={handleSave}
          />

          <DangerZoneSection
            canDelete={canDelete}
            permissionsLoading={permissionsLoading}
            onOpenDelete={() => setDeleteOpen(true)}
          />
        </div>
      </div>

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
  nameError: string;
  canUpdate: boolean;
  permissionsLoading: boolean;
  isSaving: boolean;
  isDirty: boolean;
  onNameChange: (next: string) => void;
  onDescriptionChange: (next: string) => void;
  onSave: () => Promise<void> | void;
}

function WorkspaceDetailsSection({
  name,
  description,
  nameError,
  canUpdate,
  permissionsLoading,
  isSaving,
  isDirty,
  onNameChange,
  onDescriptionChange,
  onSave,
}: WorkspaceDetailsSectionProps) {
  return (
    <section className={cn("p-6", factoryCardClassName)} data-testid="factory-settings-general-form">
      <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Workspace details</h2>
      <div className="mt-4 space-y-4">
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
      </div>
      <div className="mt-6 flex justify-end">
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
      </div>
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
    <section className="rounded-lg border border-destructive/40 bg-card p-6" data-testid="factory-settings-danger-zone">
      <h2 className="text-[13px] font-medium tracking-[-0.01em] text-destructive">Danger zone</h2>
      <p className="mt-1 text-[13px] text-muted-foreground">
        Deleting a workspace permanently removes its work orders, lines, and apps.
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
    </section>
  );
}
