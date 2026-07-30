import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useCreateFactory, useFactories } from "@/hooks/useFactoryData";
import { cn } from "@/lib/utils";
import { Factory as FactoryIcon } from "lucide-react";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { homeListCardClassName, homePageSubtitleClassName, homePageTitleClassName } from "../home/homePageStyles";
import { CreateFactoryDialog } from "./CreateFactoryDialog";
import { FactoryPageShell } from "./FactoryPageShell";

export function FactoryListPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [createOpen, setCreateOpen] = useState(false);

  usePageTitle(["Factories"]);

  const { data: factories = [], isLoading, isFetching, error } = useFactories(organizationId ?? "");
  const createFactory = useCreateFactory(organizationId ?? "");

  const canCreate = canAct("factories", "create");
  const isPageLoading = isLoading || (isFetching && factories.length === 0);

  useReportPageReady(!isPageLoading && Boolean(organizationId), {
    factory_count: factories.length,
    failed: Boolean(error),
  });

  if (!organizationId) {
    return null;
  }

  const handleCreate = async (input: { name: string; description: string }) => {
    const factory = await createFactory.mutateAsync(input);
    setCreateOpen(false);
    if (factory.id) {
      navigate(`/${organizationId}/factories/${factory.id}`);
    }
  };

  return (
    <FactoryPageShell>
      <div className="mx-auto w-full max-w-6xl p-8">
        <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
          <div>
            <Heading className={homePageTitleClassName}>Factories</Heading>
            <Text className={homePageSubtitleClassName}>
              Manage software production work — work orders, sources, and agents.
            </Text>
          </div>
          <PermissionTooltip
            allowed={canCreate || permissionsLoading}
            message="You don't have permission to create factories."
          >
            <Button
              type="button"
              onClick={() => setCreateOpen(true)}
              disabled={!canCreate}
              data-testid="factory-list-create-button"
            >
              <Icon name="plus" />
              Create factory
            </Button>
          </PermissionTooltip>
        </div>

        {error ? (
          <div className="rounded border border-red-300 bg-white px-4 py-2 text-red-500 dark:border-red-800 dark:bg-gray-800 dark:text-red-400">
            <Text>Failed to load factories.</Text>
          </div>
        ) : isPageLoading ? (
          <Text className="text-sm text-gray-500">Loading factories…</Text>
        ) : factories.length === 0 ? (
          <EmptyState
            canCreate={canCreate}
            permissionsLoading={permissionsLoading}
            onCreateClick={() => setCreateOpen(true)}
          />
        ) : (
          <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {factories.map((factory) => (
              <li key={factory.id}>
                <Link
                  href={`/${organizationId}/factories/${factory.id}`}
                  className={cn(
                    homeListCardClassName,
                    "block p-4 no-underline hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400",
                  )}
                >
                  <div className="flex items-start gap-3">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-300">
                      <FactoryIcon className="h-5 w-5" aria-hidden />
                    </div>
                    <div className="min-w-0">
                      <p className="truncate font-medium text-slate-900 dark:text-gray-100">{factory.name}</p>
                      {factory.description ? (
                        <p className="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400">
                          {factory.description}
                        </p>
                      ) : null}
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>

      <CreateFactoryDialog
        open={createOpen}
        isSaving={createFactory.isPending}
        onClose={() => setCreateOpen(false)}
        onCreate={handleCreate}
      />
    </FactoryPageShell>
  );
}

function EmptyState({
  canCreate,
  permissionsLoading,
  onCreateClick,
}: {
  canCreate: boolean;
  permissionsLoading: boolean;
  onCreateClick: () => void;
}) {
  return (
    <div className="flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white px-6 py-12 text-center dark:border-gray-700 dark:bg-gray-900">
      <FactoryIcon className="h-10 w-10 text-slate-400 dark:text-gray-500" aria-hidden />
      <p className="mt-4 text-base font-medium text-slate-900 dark:text-gray-100">Create your first factory</p>
      <p className="mt-2 max-w-md text-sm text-gray-500 dark:text-gray-400">
        Factories are where work orders live — intake, triage, planning, and agent delegation.
      </p>
      <PermissionTooltip
        allowed={canCreate || permissionsLoading}
        message="You don't have permission to create factories."
      >
        <Button type="button" className="mt-6" onClick={onCreateClick} disabled={!canCreate}>
          <Icon name="plus" />
          Create factory
        </Button>
      </PermissionTooltip>
    </div>
  );
}
