import type { FactoriesFactoryLine } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { CornerDownRight, MoveRight } from "lucide-react";
import { factoryCountBadgeClassName } from "./lib/factoryPageStyles";

interface FactoryLinesSidebarProps {
  factoryHref: string;
  lines: FactoriesFactoryLine[];
  canUpdate: boolean;
}

export function FactoryLinesSidebar({ factoryHref, lines, canUpdate }: FactoryLinesSidebarProps) {
  const createLineHref = `${factoryHref}/lines/new`;

  return (
    <section
      className="space-y-4 border-t border-gray-200 pt-6 dark:border-gray-700/70"
      data-testid="factory-lines-sidebar"
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Lines</h2>
          <span className={factoryCountBadgeClassName}>{lines.length}</span>
        </div>
        {canUpdate ? (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            asChild
            className="rounded-md text-gray-700 dark:text-gray-300"
            data-testid="factory-lines-sidebar-create"
          >
            <Link href={createLineHref}>+ New</Link>
          </Button>
        ) : null}
      </div>

      {lines.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">No lines yet.</p>
      ) : (
        <ul className="space-y-5">
          {lines.map((line) => {
            const editHref = line.id ? `${factoryHref}/lines/${line.id}/edit` : factoryHref;

            return (
              <li key={line.id} data-testid={`factory-line-sidebar-${line.name}`}>
                {canUpdate && line.id ? (
                  <Link
                    href={editHref}
                    className="text-sm font-medium text-gray-900 no-underline hover:text-gray-600 dark:text-gray-100 dark:hover:text-gray-300"
                    data-testid={`factory-line-edit-${line.name}`}
                  >
                    {line.name}
                  </Link>
                ) : (
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{line.name}</p>
                )}

                {(line.steps?.length ?? 0) > 0 ? (
                  <ul className="mt-2 space-y-1">
                    {line.steps!.map((step, index) => (
                      <li key={`${line.id}-${step.name}-${index}`} className="flex items-center gap-2 text-sm">
                        {index === 0 ? (
                          <MoveRight className="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden />
                        ) : (
                          <CornerDownRight
                            className="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500"
                            aria-hidden
                          />
                        )}
                        <span className="text-gray-600 dark:text-gray-300">{step.name || "Unnamed step"}</span>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
