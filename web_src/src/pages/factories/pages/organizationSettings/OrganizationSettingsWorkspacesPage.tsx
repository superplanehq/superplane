import type { FactoriesFactory } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useDeleteFactory, useFactories } from "@/hooks/useFactoryData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Settings, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link, useParams } from "react-router";
import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { factoryHomePath, factorySettingsWorkspaceGeneralPath, firstFactoryLineId } from "../../lib/factoryPagePaths";
import { clearLastVisitedFactory } from "../../lib/lastVisitedFactory";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

export function OrganizationSettingsWorkspacesPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data: organization } = useOrganization(organizationId || "");
  const { data: factories = [], isLoading, error } = useFactories(organizationId || "");
  const organizationName = organization?.metadata?.name || "Organization";

  usePageTitle(["Workspaces", organizationName]);

  if (!organizationId) {
    return null;
  }

  return (
    <FactorySettingsPageFrame title="Workspaces" subtitle="See the workspaces in this organization.">
      <FactorySettingsCard data-testid="organization-settings-workspaces">
        <WorkspacesBody organizationId={organizationId} isLoading={isLoading} error={error} factories={factories} />
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}

function WorkspacesBody({
  organizationId,
  isLoading,
  error,
  factories,
}: {
  organizationId: string;
  isLoading: boolean;
  error: unknown;
  factories: FactoriesFactory[];
}) {
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const deleteFactory = useDeleteFactory(organizationId);
  const [pendingDelete, setPendingDelete] = useState<FactoriesFactory | null>(null);

  const canOpenSettings = canAct("factories", "update");
  const canDelete = canAct("factories", "delete");

  const handleDelete = async () => {
    if (!pendingDelete?.id) {
      return;
    }
    try {
      await deleteFactory.mutateAsync(pendingDelete.id);
      clearLastVisitedFactory(account?.id ?? "", organizationId, pendingDelete.id);
      showSuccessToast("Workspace deleted.");
    } catch (deleteError) {
      showErrorToast(getApiErrorMessage(deleteError, "Failed to delete workspace."));
      throw new Error("Failed to delete workspace", { cause: deleteError });
    }
  };

  if (isLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading workspaces…</p>;
  }

  if (error) {
    return <p className="text-[13px] text-destructive">Failed to load workspaces.</p>;
  }

  if (factories.length === 0) {
    return <p className="text-[13px] text-muted-foreground">This organization has no workspaces.</p>;
  }

  return (
    <>
      <table className="w-full text-left" data-testid="organization-settings-workspaces-table">
        <thead>
          <tr className="text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
            <th className="pb-2 pr-4 font-medium">Name</th>
            <th className="w-10 pb-2 font-medium">
              <span className="sr-only">Settings</span>
            </th>
            <th className="w-10 pb-2 font-medium">
              <span className="sr-only">Delete</span>
            </th>
          </tr>
        </thead>
        <tbody className="text-[13px] text-foreground">
          {factories.map((factory) => (
            <WorkspaceRow
              key={factory.id}
              organizationId={organizationId}
              factory={factory}
              canOpenSettings={canOpenSettings}
              canDelete={canDelete}
              permissionsLoading={permissionsLoading}
              onDelete={() => setPendingDelete(factory)}
            />
          ))}
        </tbody>
      </table>

      <FactoryDeleteDialog
        open={pendingDelete !== null}
        factoryName={pendingDelete?.name ?? ""}
        title="Do you really want to delete the workspace?"
        description="This permanently removes the workspace and its work orders, lines, and apps."
        canDelete={canDelete}
        isDeleting={deleteFactory.isPending}
        onClose={() => setPendingDelete(null)}
        onConfirm={handleDelete}
      />
    </>
  );
}

function WorkspaceRow({
  organizationId,
  factory,
  canOpenSettings,
  canDelete,
  permissionsLoading,
  onDelete,
}: {
  organizationId: string;
  factory: FactoriesFactory;
  canOpenSettings: boolean;
  canDelete: boolean;
  permissionsLoading: boolean;
  onDelete: () => void;
}) {
  const name = factory.name?.trim() || "Workspace";
  const factoryKey = factory.key ?? "";
  const homeHref = factoryKey ? factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory)) : undefined;
  const settingsHref = factoryKey ? factorySettingsWorkspaceGeneralPath(organizationId, factoryKey) : undefined;

  return (
    <tr data-testid={`organization-settings-workspace-${factory.id}`}>
      <td className="py-1.5 pr-4">
        {homeHref ? (
          <Link
            to={homeHref}
            className="font-medium text-foreground underline-offset-4 hover:underline"
            data-testid={`organization-settings-workspace-name-${factory.id}`}
          >
            {name}
          </Link>
        ) : (
          <span data-testid={`organization-settings-workspace-name-${factory.id}`}>{name}</span>
        )}
      </td>
      <td className="py-1.5 text-right">
        <PermissionTooltip
          allowed={canOpenSettings || permissionsLoading}
          message="You do not have permission to open workspace settings."
        >
          {settingsHref && canOpenSettings ? (
            <Button variant="ghost" size="icon-xs" asChild>
              <Link
                to={settingsHref}
                aria-label={`Open settings for ${name}`}
                data-testid={`organization-settings-workspace-settings-${factory.id}`}
              >
                <Settings className="h-4 w-4" aria-hidden />
              </Link>
            </Button>
          ) : (
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              disabled
              aria-label={`Open settings for ${name}`}
              data-testid={`organization-settings-workspace-settings-${factory.id}`}
            >
              <Settings className="h-4 w-4" aria-hidden />
            </Button>
          )}
        </PermissionTooltip>
      </td>
      <td className="py-1.5 text-right">
        <PermissionTooltip
          allowed={canDelete || permissionsLoading}
          message="You do not have permission to delete workspaces."
        >
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            disabled={!canDelete}
            aria-label={`Delete ${name}`}
            data-testid={`organization-settings-workspace-delete-${factory.id}`}
            onClick={onDelete}
          >
            <Trash2 className="h-4 w-4" aria-hidden />
          </Button>
        </PermissionTooltip>
      </td>
    </tr>
  );
}
