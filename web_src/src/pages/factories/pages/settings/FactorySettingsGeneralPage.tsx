import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Switch } from "@/components/ui/switch";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useDeleteFactory, useSetFactoryVisibility, useUpdateFactory } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import {
  factoryLineDetailPath,
  factoryListPath,
  factorySettingsGeneralPathAfterKeyChange,
  firstFactoryLineId,
} from "../../lib/factoryPagePaths";
import { clearLastVisitedFactory } from "../../lib/lastVisitedFactory";
import {
  WORKSPACE_KEY_MAX_LENGTH,
  WORKSPACE_KEY_MIN_LENGTH,
  isValidWorkspaceKey,
  normalizeWorkspaceKey,
} from "../../lib/workspaceKey";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { PublicBadgeSettingsSection } from "./PublicBadgeSettingsSection";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { SettingsIdentityField } from "./settingsIdentityField";

const MAX_NAME_LENGTH = 128;

export function FactorySettingsGeneralPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { account } = useAccount();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const deleteFactory = useDeleteFactory(organizationId);

  usePageTitle(["General", "Settings", factory.name ?? "Workspace"]);

  const [name, setName] = useState(factory.name ?? "");
  const [key, setKey] = useState(factory.key ?? "");
  const [nameError, setNameError] = useState("");
  const [keyError, setKeyError] = useState("");
  const [deleteOpen, setDeleteOpen] = useState(false);

  useEffect(() => {
    setName(factory.name ?? "");
    setKey(factory.key ?? "");
    setNameError("");
    setKeyError("");
  }, [factory.name, factory.key]);

  const canUpdate = canAct("factories", "update");
  const canDelete = canAct("factories", "delete");
  const canPublish = canAct("factories", "publish");
  const savedName = factory.name ?? "";
  const savedKey = factory.key ?? "";
  const isDirty = name.trim() !== savedName || key !== savedKey;

  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError("Name is required.");
      return;
    }
    if (!isValidWorkspaceKey(key)) {
      setKeyError(`Use ${WORKSPACE_KEY_MIN_LENGTH} to ${WORKSPACE_KEY_MAX_LENGTH} uppercase letters.`);
      return;
    }
    try {
      const nextSettingsPath = factorySettingsGeneralPathAfterKeyChange(organizationId, savedKey, key);
      await updateFactory.mutateAsync({
        name: trimmedName,
        ...(key !== savedKey ? { key } : {}),
      });
      showSuccessToast("Workspace updated.");
      if (nextSettingsPath) {
        navigate(nextSettingsPath, { replace: true });
      }
    } catch (error) {
      const message = getApiErrorMessage(error, "Failed to update workspace");
      if (message.toLowerCase().includes("workspace key") || message.toLowerCase().includes("slug")) {
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
      <FactorySettingsPageFrame title="General" subtitle="Name and slug for this workspace.">
        <WorkspaceDetailsSection
          name={name}
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
          onKeyChange={(next) => {
            setKey(normalizeWorkspaceKey(next));
            if (keyError) setKeyError("");
          }}
          onSave={handleSave}
        />

        <VisibilitySection
          organizationId={organizationId}
          factoryId={factoryId}
          isPublic={Boolean(factory.public)}
          publicBoardPath={publicBoardPath(organizationId, factory)}
          canPublish={canPublish}
          permissionsLoading={permissionsLoading}
        />

        <PublicBadgeSettingsSection
          factory={factory}
          publicBoardPath={publicBoardPath(organizationId, factory)}
          canUpdate={canUpdate}
          permissionsLoading={permissionsLoading}
          isSaving={updateFactory.isPending}
          onUpdate={(input) => updateFactory.mutateAsync(input)}
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
  factoryKey: string;
  nameError: string;
  keyError: string;
  canUpdate: boolean;
  permissionsLoading: boolean;
  isSaving: boolean;
  isDirty: boolean;
  onNameChange: (next: string) => void;
  onKeyChange: (next: string) => void;
  onSave: () => Promise<void> | void;
}

function WorkspaceDetailsSection({
  name,
  factoryKey,
  nameError,
  keyError,
  canUpdate,
  permissionsLoading,
  isSaving,
  isDirty,
  onNameChange,
  onKeyChange,
  onSave,
}: WorkspaceDetailsSectionProps) {
  return (
    <FactorySettingsCard
      title="Workspace information"
      data-testid="factory-settings-general-form"
      className="scroll-mt-8"
      id="factory-settings-general-form"
    >
      <div className="space-y-6">
        <SettingsIdentityField
          name={name}
          nameId="factory-settings-name"
          nameTestId="factory-settings-name"
          avatarTestId="factory-settings-workspace-avatar"
          maxLength={MAX_NAME_LENGTH}
          disabled={!canUpdate}
          error={nameError}
          helperText="This name appears in the sidebar and workspace switcher."
          onNameChange={onNameChange}
        />

        <div className="space-y-2 scroll-mt-8" id="factory-settings-key">
          <Label htmlFor="factory-settings-slug">Slug</Label>
          <Input
            id="factory-settings-slug"
            data-testid="factory-settings-key"
            value={factoryKey}
            onChange={(event) => onKeyChange(event.target.value)}
            maxLength={WORKSPACE_KEY_MAX_LENGTH}
            disabled={!canUpdate}
            className="max-w-xs uppercase tracking-wider"
            autoComplete="off"
          />
          <p className="text-[12px] text-muted-foreground">
            Changing the slug updates every task identifier for this workspace. IDs already shared elsewhere will no
            longer resolve.
          </p>
          {keyError ? <p className="text-[11px] text-destructive">{keyError}</p> : null}
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
      </div>
    </FactorySettingsCard>
  );
}

function publicBoardPath(
  organizationId: string,
  factory: { key?: string; lines?: Array<{ id?: string }> | null; public?: boolean },
): string | undefined {
  if (!factory.public || !factory.key) {
    return undefined;
  }
  const lineId = firstFactoryLineId(factory);
  if (!lineId) {
    return undefined;
  }
  return factoryLineDetailPath(organizationId, factory.key, lineId);
}

interface VisibilitySectionProps {
  organizationId: string;
  factoryId: string;
  isPublic: boolean;
  publicBoardPath?: string;
  canPublish: boolean;
  permissionsLoading: boolean;
}

function VisibilitySection({
  organizationId,
  factoryId,
  isPublic,
  publicBoardPath,
  canPublish,
  permissionsLoading,
}: VisibilitySectionProps) {
  const setVisibility = useSetFactoryVisibility(organizationId, factoryId);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [nextPublic, setNextPublic] = useState(isPublic);
  const actionLabel = isPublic ? "Change to private" : "Change to public";

  const requestChange = (checked: boolean) => {
    if (checked === isPublic || setVisibility.isPending) {
      return;
    }
    setNextPublic(checked);
    setConfirmOpen(true);
  };

  const confirmChange = async () => {
    try {
      await setVisibility.mutateAsync(nextPublic);
      showSuccessToast(nextPublic ? "Workspace is public." : "Workspace is private.");
      setConfirmOpen(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update workspace visibility"));
    }
  };

  return (
    <FactorySettingsCard title="Visibility" data-testid="factory-settings-visibility-card">
      <div className="flex items-start justify-between gap-6">
        <div className="min-w-0 space-y-0.5">
          <p className="text-[13px] font-medium text-foreground">{isPublic ? "Public" : "Private"}</p>
          <p className="text-[12px] leading-5 text-muted-foreground">
            {isPublic
              ? "Anyone with the link can open the line board. Tasks, logs, and settings stay private."
              : "Only members of this organization can open the line board."}
            {isPublic && publicBoardPath ? (
              <>
                {" "}
                <Link href={publicBoardPath} className="underline" data-testid="factory-settings-visibility-board-link">
                  View public board
                </Link>
              </>
            ) : null}
          </p>
        </div>
        <PermissionTooltip
          allowed={canPublish || permissionsLoading}
          message="You do not have permission to change workspace visibility."
        >
          <Switch
            checked={isPublic}
            disabled={!canPublish || setVisibility.isPending}
            onCheckedChange={requestChange}
            aria-label={actionLabel}
            className="mt-0.5"
            data-testid="factory-settings-visibility"
          />
        </PermissionTooltip>
      </div>
      <AlertDialog
        open={confirmOpen}
        onOpenChange={(open) => {
          if (!setVisibility.isPending) {
            setConfirmOpen(open);
          }
        }}
      >
        <AlertDialogContent data-testid="factory-settings-visibility-dialog">
          <AlertDialogHeader>
            <AlertDialogTitle>
              {nextPublic ? "Make this workspace public?" : "Make this workspace private?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {nextPublic
                ? "Anyone with the link can open the line board. Tasks, logs, and settings stay private."
                : "Only members of this organization can open the line board."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={setVisibility.isPending} data-testid="factory-settings-visibility-cancel">
              {nextPublic ? "Keep private" : "Keep public"}
            </AlertDialogCancel>
            <LoadingButton
              loading={setVisibility.isPending}
              loadingText="Saving..."
              onClick={() => void confirmChange()}
              data-testid="factory-settings-visibility-confirm"
            >
              {nextPublic ? "Make public" : "Make private"}
            </LoadingButton>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FactorySettingsCard>
  );
}

interface DangerZoneSectionProps {
  canDelete: boolean;
  permissionsLoading: boolean;
  onOpenDelete: () => void;
}

function DangerZoneSection({ canDelete, permissionsLoading, onOpenDelete }: DangerZoneSectionProps) {
  return (
    <FactorySettingsCard title="Danger zone" data-testid="factory-settings-danger-zone">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0 space-y-0.5">
          <p className="text-[13px] font-medium text-foreground">Delete workspace</p>
          <p className="text-[12px] text-muted-foreground">
            SuperPlane permanently removes the tasks, lines, and apps in this workspace.
          </p>
        </div>
        <PermissionTooltip
          allowed={canDelete || permissionsLoading}
          message="You don't have permission to delete workspaces."
        >
          <Button
            type="button"
            size="sm"
            variant="destructive"
            disabled={!canDelete}
            onClick={onOpenDelete}
            data-testid="factory-settings-delete-button"
          >
            Delete workspace
          </Button>
        </PermissionTooltip>
      </div>
    </FactorySettingsCard>
  );
}
