import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp } from "lucide-react";

import { Avatar } from "@/components/Avatar/avatar";
import { getUserInitials } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";

import { formatDurationHours } from "../lib/factoryVelocityFlow";
import type { VelocityPerson } from "../lib/factoryVelocityReport";

type SortKey = "total" | "authoredMerged" | "factoryMerged" | "factoryWaste" | "medianCycleHours" | "costUsd";

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
    key: "authoredMerged",
    label: "Authored",
    hint: "Pull requests the person merged themselves",
    format: (person) => String(person.authoredMerged),
  },
  {
    key: "factoryMerged",
    label: "SuperPlane merged",
    hint: "Merged pull requests from tasks this person opened",
    format: (person) => String(person.factoryMerged),
  },
  {
    key: "factoryWaste",
    label: "SuperPlane waste",
    hint: "Tasks this person opened that closed without a merge",
    format: (person) => String(person.factoryWaste),
  },
  {
    key: "medianCycleHours",
    label: "Median cycle",
    hint: "Median time from task start to close",
    format: (person) => (person.medianCycleHours > 0 ? formatDurationHours(person.medianCycleHours) : "—"),
  },
  {
    key: "costUsd",
    label: "Tracked cost",
    hint: "Tracked model spend of their tasks",
    format: (person) => `$${person.costUsd.toFixed(0)}`,
  },
  {
    key: "total",
    label: "Merged PRs",
    hint: "Authored plus SuperPlane merged",
    format: (person) => String(totalMerged(person)),
  },
];

function totalMerged(person: VelocityPerson): number {
  return person.authoredMerged + person.factoryMerged;
}

function sortValue(person: VelocityPerson, key: SortKey): number {
  if (key === "total") return totalMerged(person);
  return person[key];
}

export function VelocityPeopleTable({
  people,
  periodLabel,
  emptyAuthorship,
}: {
  people: VelocityPerson[];
  periodLabel: string;
  /** Names why the Authored column is empty, when the cohort is unavailable. */
  emptyAuthorship?: string;
}) {
  const [sortKey, setSortKey] = useState<SortKey>("total");

  const sorted = useMemo(
    () => [...people].sort((a, b) => sortValue(b, sortKey) - sortValue(a, sortKey)),
    [people, sortKey],
  );

  return (
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-people"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">People</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            What each member merged directly and through SuperPlane. Select a column to sort.
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
                  onSort={() => setSortKey(column.key)}
                />
              ))}
            </tr>
          </thead>
          <tbody>
            {sorted.map((person, index) => (
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

      <p className="mt-4 text-[12px] text-muted-foreground">
        {people.length} {people.length === 1 ? "person" : "people"} with activity in this period
      </p>
      {emptyAuthorship ? <p className="mt-1 text-[12px] text-muted-foreground">{emptyAuthorship}</p> : null}
    </section>
  );
}

function SortableHeader({ column, isActive, onSort }: { column: Column; isActive: boolean; onSort: () => void }) {
  return (
    <th scope="col" className="pb-2 pl-6 text-right text-[12px] font-normal">
      <button
        type="button"
        onClick={onSort}
        title={column.hint}
        aria-sort={isActive ? "descending" : "none"}
        className={cn(
          "inline-flex items-center gap-1 transition-colors hover:text-foreground",
          isActive ? "text-foreground" : "text-muted-foreground",
        )}
      >
        {isActive ? <ArrowDown className="size-3" aria-hidden /> : <ArrowUp className="size-3 opacity-0" aria-hidden />}
        {column.label}
      </button>
    </th>
  );
}
