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

interface SaveOrganizationIdentityInput {
  canUpdate: boolean;
  organizationId: string | undefined;
  trimmedName: string;
  trimmedSlug: string;
  currentSlug: string;
  isDirty: boolean;
  slugChanged: boolean;
  updateOrganization: (payload: { name: string; slug?: string }) => Promise<unknown>;
  pathname: string;
  search: string;
  navigate: (path: string, options: { replace: boolean }) => void;
  setNameError: (message: string) => void;
  setSlugError: (message: string | null) => void;
}

async function saveOrganizationIdentity({
  canUpdate,
  organizationId,
  trimmedName,
  trimmedSlug,
  currentSlug,
  isDirty,
  slugChanged,
  updateOrganization,
  pathname,
  search,
  navigate,
  setNameError,
  setSlugError,
}: SaveOrganizationIdentityInput) {
  if (!canUpdate || !organizationId) {
    return;
  }

  if (!trimmedName) {
    setNameError("Name is required.");
    return;
  }
  setNameError("");

  if (!isDirty) {
    return;
  }

  if (slugChanged) {
    const validationError = validateOrganizationSlug(trimmedSlug);
    if (validationError) {
      setSlugError(organizationSlugValidationMessage(validationError));
      return;
    }
  }
  setSlugError(null);

  try {
    await updateOrganization({
      name: trimmedName,
      ...(slugChanged ? { slug: trimmedSlug } : {}),
    });

    showSuccessToast("Organization updated.");

    // The URL segment only needs rewriting when it currently carries the
    // old slug. When it carries the organization ID instead, leave it
    // alone so navigation keeps working.
    if (slugChanged && organizationId === currentSlug) {
      const newPath = pathname.replace(`/${organizationId}/`, `/${trimmedSlug}/`);
      navigate(`${newPath}${search}`, { replace: true });
    }
  } catch (err) {
    const message = getApiErrorMessage(err, "Failed to update organization slug");
    setSlugError(message);
    showErrorToast(message);
  }
}

function organizationIdOrEmpty(organizationId: string | undefined) {
  return organizationId || "";
}

function organizationIdentity(organization: ReturnType<typeof useOrganization>["data"]) {
  return {
    name: organization?.metadata?.name || "",
    slug: organization?.metadata?.slug || "",
  };
}

export function OrganizationSettingsOverviewPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: organization } = useOrganization(organizationIdOrEmpty(organizationId));
  const deleteOrganizationMutation = useDeleteOrganization(organizationIdOrEmpty(organizationId));
  const { name: currentName, slug: currentSlug } = organizationIdentity(organization);
  const [deleteOpen, setDeleteOpen] = useState(false);

  usePageTitle(["General", currentName || "Organization"]);

  const canUpdateOrg = canAct("org", "update");
  const canDeleteOrg = canAct("org", "delete");

  return (
    <>
      <FactorySettingsPageFrame title="General" subtitle="Name and slug for this organization.">
        <OrganizationInformationCard
          organizationId={organizationId}
          currentName={currentName}
          currentSlug={currentSlug}
          canUpdate={canUpdateOrg}
          permissionsLoading={permissionsLoading}
        />
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

interface OrganizationInformationCardProps {
  organizationId: string | undefined;
  currentName: string;
  currentSlug: string;
  canUpdate: boolean;
  permissionsLoading: boolean;
}

function OrganizationInformationCard({
  organizationId,
  currentName,
  currentSlug,
  canUpdate,
  permissionsLoading,
}: OrganizationInformationCardProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const updateOrganizationMutation = useUpdateOrganization(organizationIdOrEmpty(organizationId));
  const [name, setName] = useState(currentName);
  const [slug, setSlug] = useState(currentSlug);
  const [nameError, setNameError] = useState("");
  const [slugError, setSlugError] = useState<string | null>(null);

  useEffect(() => {
    setName(currentName);
    setSlug(currentSlug);
    setNameError("");
    setSlugError(null);
  }, [currentName, currentSlug]);

  const trimmedName = name.trim();
  const trimmedSlug = slug.trim();
  const slugChanged = trimmedSlug !== currentSlug;
  const isDirty = trimmedName !== currentName || slugChanged;

  return (
    <FactorySettingsCard title="Organization information" data-testid="organization-settings-overview">
      <div className="space-y-6">
        <SettingsIdentityField
          name={name}
          nameId="organization-settings-overview-name"
          nameTestId="organization-settings-overview-name"
          avatarTestId="organization-settings-overview-avatar"
          maxLength={MAX_NAME_LENGTH}
          disabled={!canUpdate}
          error={nameError}
          helperText="This name appears in the sidebar and organization switcher."
          onNameChange={(next) => {
            setName(next);
            if (nameError) setNameError("");
          }}
        />
        <OrganizationSlugField
          slug={slug}
          slugError={slugError}
          disabled={!canUpdate}
          onSlugChange={(next) => {
            setSlug(next);
            setSlugError(null);
          }}
        />
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to update this organization."
        >
          <LoadingButton
            type="button"
            data-testid="organization-settings-overview-save"
            onClick={() =>
              void saveOrganizationIdentity({
                canUpdate,
                organizationId,
                trimmedName,
                trimmedSlug,
                currentSlug,
                isDirty,
                slugChanged,
                updateOrganization: (payload) => updateOrganizationMutation.mutateAsync(payload),
                pathname: location.pathname,
                search: location.search,
                navigate,
                setNameError,
                setSlugError,
              })
            }
            disabled={saveControlDisabled(canUpdate, trimmedName, isDirty)}
            loading={updateOrganizationMutation.isPending}
            loadingText="Saving..."
          >
            Save
          </LoadingButton>
        </PermissionTooltip>
      </div>
    </FactorySettingsCard>
  );
}

function saveControlDisabled(canUpdate: boolean, trimmedName: string, isDirty: boolean) {
  return !canUpdate || !trimmedName || !isDirty;
}

interface OrganizationSlugFieldProps {
  slug: string;
  slugError: string | null;
  disabled: boolean;
  onSlugChange: (slug: string) => void;
}

function OrganizationSlugField({ slug, slugError, disabled, onSlugChange }: OrganizationSlugFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="organization-settings-overview-slug">Slug</Label>
      <Input
        id="organization-settings-overview-slug"
        data-testid="organization-settings-overview-slug-input"
        value={slug}
        onChange={(event) => onSlugChange(event.target.value)}
        disabled={disabled}
      />
      <p className="text-[12px] text-muted-foreground">
        Used in your workspace URL. Use lowercase letters, numbers, and dashes only.
      </p>
      {slugError ? <p className="text-[11px] text-destructive">{slugError}</p> : null}
    </div>
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
            You lose access now. SuperPlane keeps this organization for at least 30 days, then removes it.
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
