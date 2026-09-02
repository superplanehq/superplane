import { useState } from "react";
import { Link } from "react-router";

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
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { useFactories } from "@/hooks/useFactoryData";
import { getNameInitials } from "@/lib/nameInitials";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Trash2 } from "lucide-react";

import { factorySettingsSectionPath } from "../../../lib/factoryPagePaths";
import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import {
  SettingsDangerPanel,
  SettingsIdentityHero,
  SettingsListRow,
  SettingsSaveBar,
  SettingsStackedField,
  SettingsUrlField,
} from "./settingsRedesignParts";
import {
  ORGANIZATION_NAME_MAX_LENGTH,
  useSettingsRedesignOrganizationGeneral,
} from "./useSettingsRedesignOrganizationGeneral";

export function SettingsRedesignOrganizationGeneralPage() {
  const general = useSettingsRedesignOrganizationGeneral();
  const organizationName = general.name.trim() || general.organization?.metadata?.name || "Organization";
  usePageTitle(["General", organizationName]);

  return (
    <FactorySettingsPageFrame title="General" subtitle="Name and URL for this organization.">
      <div className="space-y-8" data-testid="organization-settings-overview">
        <OrganizationIdentityForm general={general} organizationName={organizationName} />
        <OrganizationWorkspaces organizationId={general.organizationId} />
        <OrganizationDanger
          general={general}
          organizationName={general.organization?.metadata?.name || organizationName}
        />
      </div>
    </FactorySettingsPageFrame>
  );
}

function OrganizationIdentityForm({
  general,
  organizationName,
}: {
  general: ReturnType<typeof useSettingsRedesignOrganizationGeneral>;
  organizationName: string;
}) {
  const slugCaption = `/${general.slug || "slug"}`;

  return (
    <div className="space-y-6">
      <SettingsIdentityHero
        initials={getNameInitials(organizationName)}
        title={organizationName}
        caption={slugCaption}
      />

      <SettingsStackedField htmlFor="organization-settings-overview-name" label="Name">
        <Input
          id="organization-settings-overview-name"
          data-testid="organization-settings-overview-name"
          value={general.name}
          maxLength={ORGANIZATION_NAME_MAX_LENGTH}
          disabled={!general.canUpdate}
          onChange={(event) => {
            if (event.target.value.length <= ORGANIZATION_NAME_MAX_LENGTH) {
              general.setName(event.target.value);
            }
          }}
        />
      </SettingsStackedField>

      <SettingsStackedField
        htmlFor="organization-settings-overview-slug"
        label="Slug"
        hint="Used in workspace URLs. Use lowercase letters, numbers, and dashes only."
      >
        <SettingsUrlField
          prefix="/"
          id="organization-settings-overview-slug"
          testId="organization-settings-overview-slug-input"
          value={general.slug}
          disabled={!general.canUpdate}
          onChange={(value) => {
            general.setSlug(value);
            general.clearSlugError();
          }}
        />
        {general.slugError ? <p className="text-[11px] text-destructive">{general.slugError}</p> : null}
      </SettingsStackedField>

      <SettingsStackedField htmlFor="organization-settings-overview-description" label="Description">
        <Textarea
          id="organization-settings-overview-description"
          data-testid="organization-settings-overview-description"
          value={general.description}
          rows={3}
          disabled={!general.canUpdate}
          onChange={(event) => general.setDescription(event.target.value)}
        />
      </SettingsStackedField>

      <SettingsSaveBar
        allowed={general.canUpdate}
        permissionsLoading={general.permissionsLoading}
        disabled={!general.canUpdate || !general.name.trim() || !general.isDirty}
        loading={general.updateOrganization.isPending}
        onSave={() => void general.saveDetails()}
        denyMessage="You do not have permission to update this organization."
        testId="organization-settings-overview-save"
      />
    </div>
  );
}

function OrganizationWorkspaces({ organizationId }: { organizationId: string }) {
  const { data: factories = [] } = useFactories(organizationId);

  return (
    <div className="space-y-3 border-t border-border pt-6" data-testid="settings-redesign-org-workspaces">
      <h2 className="text-[13px] font-medium text-foreground">Workspaces</h2>
      {factories.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">No workspaces yet.</p>
      ) : (
        <ul>
          {factories.map((factory) => (
            <SettingsListRow
              key={factory.id}
              title={factory.name || "Workspace"}
              subtitle={factory.key ? `/${factory.key}` : undefined}
              action={
                factory.key ? (
                  <Button asChild type="button" variant="outline" size="sm">
                    <Link to={factorySettingsSectionPath(organizationId, factory.key, "workspace", "general")}>
                      Open settings
                    </Link>
                  </Button>
                ) : null
              }
            />
          ))}
        </ul>
      )}
    </div>
  );
}

function OrganizationDanger({
  general,
  organizationName,
}: {
  general: ReturnType<typeof useSettingsRedesignOrganizationGeneral>;
  organizationName: string;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <>
      <SettingsDangerPanel
        testId="settings-redesign-org-danger"
        title="Delete organization"
        description="Permanently removes workspaces, members, and settings."
        action={
          <PermissionTooltip
            allowed={general.canDelete || general.permissionsLoading}
            message="You do not have permission to delete this organization."
          >
            <Button
              type="button"
              variant="destructive"
              size="sm"
              disabled={!general.canDelete}
              onClick={() => {
                general.clearDeleteError();
                setDeleteOpen(true);
              }}
              data-testid="settings-redesign-org-delete"
            >
              <Trash2 className="size-3.5" aria-hidden />
              Delete organization
            </Button>
          </PermissionTooltip>
        }
      />
      <OrganizationDeleteDialog
        open={deleteOpen}
        organizationName={organizationName}
        canDelete={general.canDelete}
        isDeleting={general.deleteOrganization.isPending}
        deleteError={general.deleteError}
        onOpenChange={(open) => {
          setDeleteOpen(open);
          if (!open) general.clearDeleteError();
        }}
        onConfirm={() => void general.handleDelete()}
      />
    </>
  );
}

function OrganizationDeleteDialog({
  open,
  organizationName,
  canDelete,
  isDeleting,
  deleteError,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  organizationName: string;
  canDelete: boolean;
  isDeleting: boolean;
  deleteError: string | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const [confirmation, setConfirmation] = useState("");
  const canSubmit = canDelete && confirmation === organizationName;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (!next) setConfirmation("");
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete organization</DialogTitle>
          <DialogDescription>Type {organizationName} to confirm</DialogDescription>
        </DialogHeader>
        <Input
          id="settings-redesign-org-delete-confirm"
          data-testid="settings-redesign-org-delete-confirm"
          value={confirmation}
          disabled={!canDelete}
          placeholder={organizationName}
          aria-label={`Type ${organizationName} to confirm`}
          onChange={(event) => setConfirmation(event.target.value)}
        />
        {deleteError ? <p className="text-[11px] text-destructive">{deleteError}</p> : null}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Keep organization
          </Button>
          <LoadingButton
            type="button"
            variant="destructive"
            disabled={!canSubmit}
            loading={isDeleting}
            loadingText="Deleting..."
            onClick={onConfirm}
            data-testid="settings-redesign-org-delete-submit"
          >
            Delete organization
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
