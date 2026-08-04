import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateFactory, useFactories } from "@/hooks/useFactoryData";
import { cn } from "@/lib/utils";
import { CreateFactoryDialog } from "@/pages/factories/CreateFactoryDialog";
import { Factory as FactoryIcon } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { homeListCardClassName, homePageSubtitleClassName, homePageTitleClassName } from "./homePageStyles";

interface HomeFactoriesSectionProps {
  organizationId: string;
}

export function HomeFactoriesSection({ organizationId }: HomeFactoriesSectionProps) {
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [createOpen, setCreateOpen] = useState(false);

  const { data: factories = [], isLoading, isFetching, error } = useFactories(organizationId, true);
  const createFactory = useCreateFactory(organizationId);

  const canCreate = canAct("factories", "create");
  const isSectionLoading = isLoading || (isFetching && factories.length === 0);

  const handleCreate = async (input: { name: string; description: string }) => {
    const factory = await createFactory.mutateAsync(input);
    setCreateOpen(false);
    if (factory.id) {
      navigate(`/${organizationId}/factories/${factory.id}`);
    }
  };

  return (
    <section className="mb-10" data-testid="home-factories-section">
      <div className="mb-4 flex flex-wrap items-start justify-between gap-4">
        <div>
          <Heading level={2} className={cn(homePageTitleClassName, "!text-2xl mb-1")}>
            Factories
          </Heading>
          <Text className={homePageSubtitleClassName}>Work orders, lines, and factory apps.</Text>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <PermissionTooltip
            allowed={canCreate || permissionsLoading}
            message="You don't have permission to create factories."
          >
            <Button type="button" onClick={() => setCreateOpen(true)} disabled={!canCreate}>
              <Icon name="plus" />
              New Factory
            </Button>
          </PermissionTooltip>
          <Button type="button" variant="outline" onClick={() => navigate(`/${organizationId}/factories`)}>
            View all
          </Button>
        </div>
      </div>

      {error ? (
        <div className="rounded border border-red-300 bg-white px-4 py-2 text-red-500 dark:border-red-800 dark:bg-gray-800 dark:text-red-400">
          <Text>Failed to load factories.</Text>
        </div>
      ) : isSectionLoading ? (
        <Text className="text-sm text-gray-500">Loading factories…</Text>
      ) : factories.length === 0 ? (
        <div className="rounded-lg border border-dashed border-slate-300 bg-white px-6 py-10 text-center dark:border-gray-700 dark:bg-gray-900">
          <FactoryIcon className="mx-auto h-9 w-9 text-slate-400 dark:text-gray-500" aria-hidden />
          <p className="mt-3 text-sm font-medium text-slate-900 dark:text-gray-100">No factories yet</p>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Create a factory to manage work orders and production lines.
          </p>
        </div>
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

      <CreateFactoryDialog
        open={createOpen}
        isSaving={createFactory.isPending}
        onClose={() => setCreateOpen(false)}
        onCreate={handleCreate}
      />
    </section>
  );
}
