import { useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useDeleteOrganization, useOrganization, useUpdateOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { organizationSlugValidationMessage, validateOrganizationSlug } from "@/lib/organizationSlug";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { SettingsIdentityField } from "../settings/settingsIdentityField";
import { OrganizationDeleteDialog } from "./OrganizationDeleteDialog";

const MAX_NAME_LENGTH = 128;

async function deleteOrganizationAndLeave(
  canDelete: boolean,
  organizationId: string | undefined,
  deleteOrganization: () => Promise<unknown>,
) {
  if (!canDelete || !organizationId) {
    return;
  }

  try {
    await deleteOrganization();
    showSuccessToast("Organization deleted.");
    window.location.href = "/";
  } catch (err) {
    showErrorToast(getApiErrorMessage(err, "Failed to delete organization."));
    throw err;
  }
}

export function OrganizationSettingsOverviewPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: organization } = useOrganization(organizationId || "");
  const updateOrganizationMutation = useUpdateOrganization(organizationId || "");
  const deleteOrganizationMutation = useDeleteOrganization(organizationId || "");
  const currentName = organization?.metadata?.name || "";
  const currentSlug = organization?.metadata?.slug || "";

  const [name, setName] = useState(currentName);
  const [slug, setSlug] = useState(currentSlug);
  const [nameError, setNameError] = useState("");
  const [slugError, setSlugError] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  useEffect(() => {
    setName(currentName);
    setSlug(currentSlug);
    setNameError("");
    setSlugError(null);
  }, [currentName, currentSlug]);

  usePageTitle(["General", currentName || "Organization"]);

  const canUpdateOrg = canAct("org", "update");
  const canDeleteOrg = canAct("org", "delete");
  const trimmedName = name.trim();
  const trimmedSlug = slug.trim();
  const slugChanged = trimmedSlug !== currentSlug;
  const isDirty = trimmedName !== currentName || slugChanged;

  const handleSave = async () => {
    if (!canUpdateOrg || !organizationId) return;

    if (!trimmedName) {
      setNameError("Name is required.");
      return;
    }
    setNameError("");

    if (!isDirty) return;

    if (slugChanged) {
      const validationError = validateOrganizationSlug(trimmedSlug);
      if (validationError) {
        setSlugError(organizationSlugValidationMessage(validationError));
        return;
      }
    }
    setSlugError(null);

    try {
      await updateOrganizationMutation.mutateAsync({
        name: trimmedName,
        ...(slugChanged ? { slug: trimmedSlug } : {}),
      });

      showSuccessToast("Organization updated.");

      // The URL segment only needs rewriting when it currently carries the
      // old slug. When it carries the organization ID instead, leave it
      // alone so navigation keeps working.
      if (slugChanged && organizationId === currentSlug) {
        const newPath = location.pathname.replace(`/${organizationId}/`, `/${trimmedSlug}/`);
        navigate(`${newPath}${location.search}`, { replace: true });
      }
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to update organization slug");
      setSlugError(message);
      showErrorToast(message);
    }
  };

  return (
    <>
      <FactorySettingsPageFrame title="General" subtitle="Name and slug for this organization.">
        <FactorySettingsCard title="Organization information" data-testid="organization-settings-overview">
          <div className="space-y-6">
            <SettingsIdentityField
              name={name}
              nameId="organization-settings-overview-name"
              nameTestId="organization-settings-overview-name"
              avatarTestId="organization-settings-overview-avatar"
              maxLength={MAX_NAME_LENGTH}
              disabled={!canUpdateOrg}
              error={nameError}
              helperText="This name appears in the sidebar and organization switcher."
              onNameChange={(next) => {
                setName(next);
                if (nameError) setNameError("");
              }}
            />

            <div className="space-y-2">
              <Label htmlFor="organization-settings-overview-slug">Slug</Label>
              <Input
                id="organization-settings-overview-slug"
                data-testid="organization-settings-overview-slug-input"
                value={slug}
                onChange={(event) => {
                  setSlug(event.target.value);
                  setSlugError(null);
                }}
                disabled={!canUpdateOrg}
              />
              <p className="text-[12px] text-muted-foreground">
                Used in your workspace URL. Use lowercase letters, numbers, and dashes only.
              </p>
              {slugError ? <p className="text-[11px] text-destructive">{slugError}</p> : null}
            </div>

            <PermissionTooltip
              allowed={canUpdateOrg || permissionsLoading}
              message="You don't have permission to update this organization."
            >
              <LoadingButton
                type="button"
                data-testid="organization-settings-overview-save"
                onClick={() => void handleSave()}
                disabled={!canUpdateOrg || !trimmedName || !isDirty}
                loading={updateOrganizationMutation.isPending}
                loadingText="Saving..."
              >
                Save
              </LoadingButton>
            </PermissionTooltip>
          </div>
        </FactorySettingsCard>

        <OrganizationDangerZone
          canDelete={canDeleteOrg}
          permissionsLoading={permissionsLoading}
          onOpenDelete={() => setDeleteOpen(true)}
        />
      </FactorySettingsPageFrame>

      <OrganizationDeleteDialog
        open={deleteOpen}
        organizationName={currentName}
        canDelete={canDeleteOrg}
        isDeleting={deleteOrganizationMutation.isPending}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() =>
          deleteOrganizationAndLeave(canDeleteOrg, organizationId, () => deleteOrganizationMutation.mutateAsync())
        }
      />
    </>
  );
}

interface OrganizationDangerZoneProps {
  canDelete: boolean;
  permissionsLoading: boolean;
  onOpenDelete: () => void;
}

function OrganizationDangerZone({ canDelete, permissionsLoading, onOpenDelete }: OrganizationDangerZoneProps) {
  return (
    <FactorySettingsCard title="Danger zone" data-testid="organization-settings-danger-zone">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0 space-y-0.5">
          <p className="text-[13px] font-medium text-foreground">Delete organization</p>
          <p className="text-[12px] text-muted-foreground">
            SuperPlane permanently removes all workspaces, members, and settings in this organization.
          </p>
        </div>
        <PermissionTooltip
          allowed={canDelete || permissionsLoading}
          message="You do not have permission to delete this organization."
        >
          <Button
            type="button"
            size="sm"
            variant="destructive"
            disabled={!canDelete}
            onClick={onOpenDelete}
            data-testid="organization-settings-delete-button"
          >
            Delete organization
          </Button>
        </PermissionTooltip>
      </div>
    </FactorySettingsCard>
  );
}
