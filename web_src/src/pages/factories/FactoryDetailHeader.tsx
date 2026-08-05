import type { FactoriesFactory } from "@/api-client";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { EditFactoryDialog } from "./EditFactoryDialog";
import { FactoryActionsMenu } from "./FactoryActionsMenu";
import { FactoryDeleteDialog } from "./FactoryDeleteDialog";

interface FactoryDetailHeaderProps {
  factory: FactoriesFactory;
  workOrdersCount: number;
  canCreate: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  permissionsLoading: boolean;
  createHref: string;
  isUpdating: boolean;
  isDeleting: boolean;
  onUpdate: (input: { name: string; description: string }) => Promise<void>;
  onDelete: () => Promise<void>;
}

export function FactoryDetailHeader({
  factory,
  workOrdersCount: _workOrdersCount,
  canCreate,
  canUpdate,
  canDelete,
  permissionsLoading,
  createHref,
  isUpdating,
  isDeleting,
  onUpdate,
  onDelete,
}: FactoryDetailHeaderProps) {
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h1 className="min-w-0 text-3xl font-semibold tracking-tight text-gray-900 dark:text-gray-100">
              {factory.name}
            </h1>
            <FactoryActionsMenu
              permissions={{ canUpdate, canDelete, loading: permissionsLoading }}
              isDeleting={isDeleting}
              onEdit={() => setEditOpen(true)}
              onDelete={() => setDeleteOpen(true)}
            />
          </div>
          {factory.description?.trim() ? (
            <p className="mt-3 max-w-3xl text-sm leading-relaxed text-gray-500 dark:text-gray-400">
              {factory.description.trim()}
            </p>
          ) : null}
        </div>

        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create work orders."
        >
          <Button
            type="button"
            asChild
            disabled={!canCreate}
            className={cn(appDarkModeClasses.primaryAction)}
            data-testid="work-order-list-create-button"
          >
            <Link href={createHref}>
              <Icon name="plus" />
              New Work Order
            </Link>
          </Button>
        </PermissionTooltip>
      </div>

      <EditFactoryDialog
        open={editOpen}
        initialName={factory.name ?? ""}
        initialDescription={factory.description ?? ""}
        isSaving={isUpdating}
        onClose={() => setEditOpen(false)}
        onSave={async (input) => {
          await onUpdate(input);
          setEditOpen(false);
        }}
      />

      <FactoryDeleteDialog
        open={deleteOpen}
        factoryName={factory.name ?? ""}
        canDelete={canDelete}
        isDeleting={isDeleting}
        onClose={() => setDeleteOpen(false)}
        onConfirm={onDelete}
      />
    </>
  );
}
