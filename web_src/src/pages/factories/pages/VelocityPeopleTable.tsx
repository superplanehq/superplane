import { ArrowDown, ArrowUp, Loader2 } from "lucide-react";

import { Avatar } from "@/components/Avatar/avatar";
import { Button } from "@/components/ui/button";
import { getUserInitials } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";

import { formatDurationHours } from "../lib/factoryVelocityFlow";
import type { VelocityPerson } from "../lib/factoryVelocityReport";
import type { PeopleSortDirection, PeopleSortKey } from "../lib/velocityPeopleSort";

type SortKey = PeopleSortKey;

/** Members without a connected account photo still need a stable, legible avatar. */
const AVATAR_COLORS = [
  "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
  "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300",
  "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
  "bg-violet-100 text-violet-700 dark:bg-violet-950 dark:text-violet-300",
  "bg-rose-100 text-rose-700 dark:bg-rose-950 dark:text-rose-300",
  "bg-cyan-100 text-cyan-700 dark:bg-cyan-950 dark:text-cyan-300",
];

function avatarColorClass(id: string): string {
  let hash = 0;
  for (let index = 0; index < id.length; index += 1) {
    hash = (hash * 31 + id.charCodeAt(index)) % AVATAR_COLORS.length;
  }
  return AVATAR_COLORS[hash] ?? AVATAR_COLORS[0]!;
}

interface Column {
  key: SortKey;
  label: string;
  hint: string;
  format: (person: VelocityPerson) => string;
}

const COLUMNS: Column[] = [
  {
    key: "factoryMerged",
    label: "Via SuperPlane",
    hint: "Merged pull requests from SuperPlane tasks this person opened",
    format: (person) => String(person.factoryMerged),
  },
  {
    key: "authoredMerged",
    label: "Manual work",
    hint: "Pull requests this person created without SuperPlane",
    format: (person) => String(person.authoredMerged),
  },
  {
    key: "medianCycleHours",
    label: "Median cycle",
    hint: "Median time from task start to close",
    format: (person) => (person.medianCycleHours > 0 ? formatDurationHours(person.medianCycleHours) : "—"),
  },
  {
    key: "costUsd",
    label: "Costs",
    hint: "Tracked model spend of their SuperPlane tasks",
    format: (person) => `$${person.costUsd.toFixed(0)}`,
  },
  {
    key: "total",
    label: "Merged PRs",
    hint: "Via SuperPlane plus manual work",
    format: (person) => String(totalMerged(person)),
  },
];

function totalMerged(person: VelocityPerson): number {
  return person.authoredMerged + person.factoryMerged;
}

export function VelocityPeopleTable({
  people,
  total,
  periodLabel,
  emptyAuthorship,
  sortKey,
  sortDirection,
  onSort,
  canLoadMore,
  isLoadingMore,
  onLoadMore,
}: {
  /** Rows fetched so far, already sorted and paged by the backend. */
  people: VelocityPerson[];
  /** Total people with activity in the period, across every page. */
  total: number;
  periodLabel: string;
  /** Names why the Manual work column is empty, when the cohort is unavailable. */
  emptyAuthorship?: string;
  sortKey: SortKey;
  sortDirection: PeopleSortDirection;
  /** Sorts by `key`. The caller toggles direction when `key` is already active. */
  onSort: (key: SortKey) => void;
  canLoadMore: boolean;
  isLoadingMore: boolean;
  onLoadMore: () => void;
}) {
  return (
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-people"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">People</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            {total} {total === 1 ? "person" : "people"} with activity in this period
          </p>
        </div>
        <p className="text-[12px] text-muted-foreground">{periodLabel}</p>
      </div>

      <div className="mt-5 overflow-x-auto">
        <table className="w-full min-w-[720px] border-collapse text-[13px]">
          <thead>
            <tr className="border-b border-border">
              <th scope="col" className="w-8 pb-2 text-left text-[12px] font-normal text-muted-foreground">
                <span className="sr-only">Rank</span>
              </th>
              <th scope="col" className="pb-2 text-left text-[12px] font-normal text-muted-foreground">
                Member
              </th>
              {COLUMNS.map((column) => (
                <SortableHeader
                  key={column.key}
                  column={column}
                  isActive={sortKey === column.key}
                  direction={sortDirection}
                  onSort={() => onSort(column.key)}
                />
              ))}
            </tr>
          </thead>
          <tbody>
            {people.map((person, index) => (
              <tr key={person.id} className="border-b border-border/60 last:border-b-0">
                <td className="py-3 text-[12px] tabular-nums text-muted-foreground">{index + 1}</td>
                <td className="py-3 pr-6">
                  <div className="flex min-w-0 items-center gap-2.5">
                    <Avatar
                      src={person.avatarUrl}
                      initials={getUserInitials(person.name)}
                      alt={person.name}
                      className={cn("size-7 shrink-0", avatarColorClass(person.id))}
                    />
                    <div className="min-w-0">
                      <p className="truncate text-foreground">{person.name}</p>
                      <p className="truncate text-[12px] text-muted-foreground">{person.email}</p>
                    </div>
                  </div>
                </td>
                {COLUMNS.map((column) => (
                  <td
                    key={column.key}
                    className={cn(
                      "py-3 pl-6 text-right tabular-nums",
                      sortKey === column.key ? "text-foreground" : "text-muted-foreground",
                    )}
                  >
                    {column.format(person)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {canLoadMore ? (
        <div className="mt-4 flex flex-col items-center gap-1.5">
          <Button type="button" variant="outline" size="sm" onClick={onLoadMore} disabled={isLoadingMore}>
            {isLoadingMore ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
            Load more
          </Button>
          <p className="text-[12px] text-muted-foreground">
            Showing {people.length} of {total}
          </p>
        </div>
      ) : null}

      {emptyAuthorship ? <p className="mt-4 text-[12px] text-muted-foreground">{emptyAuthorship}</p> : null}
    </section>
  );
}

function SortableHeader({
  column,
  isActive,
  direction,
  onSort,
}: {
  column: Column;
  isActive: boolean;
  direction: PeopleSortDirection;
  onSort: () => void;
}) {
  const ariaSort = isActive ? (direction === "asc" ? "ascending" : "descending") : "none";
  const Icon = isActive && direction === "asc" ? ArrowUp : ArrowDown;

  return (
    <th scope="col" className="pb-2 pl-6 text-right text-[12px] font-normal">
      <button
        type="button"
        onClick={onSort}
        title={column.hint}
        aria-sort={ariaSort}
        className={cn(
          "inline-flex items-center gap-1 transition-colors hover:text-foreground",
          isActive ? "text-foreground" : "text-muted-foreground",
        )}
      >
        <Icon className={cn("size-3", !isActive && "opacity-0")} aria-hidden />
        {column.label}
      </button>
    </th>
  );
}
