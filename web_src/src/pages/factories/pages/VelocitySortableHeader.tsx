import { ArrowDown, ArrowUp } from "lucide-react";

import { cn } from "@/lib/utils";

export type VelocitySortDirection = "asc" | "desc";

/**
 * A right-aligned metric column header that sorts the table it belongs to.
 *
 * The arrow keeps its space when the column is inactive, so the labels do not
 * shift sideways as the reader moves the sort from column to column.
 */
export function VelocitySortableHeader({
  label,
  hint,
  isActive,
  direction,
  onSort,
}: {
  label: string;
  hint: string;
  isActive: boolean;
  direction: VelocitySortDirection;
  onSort: () => void;
}) {
  const ariaSort = isActive ? (direction === "asc" ? "ascending" : "descending") : "none";
  const Icon = isActive && direction === "asc" ? ArrowUp : ArrowDown;

  return (
    <th scope="col" className="pb-2 pl-6 text-right text-[12px] font-normal">
      <button
        type="button"
        onClick={onSort}
        title={hint}
        aria-sort={ariaSort}
        className={cn(
          "inline-flex items-center gap-1 transition-colors hover:text-foreground",
          isActive ? "text-foreground" : "text-muted-foreground",
        )}
      >
        <Icon className={cn("size-3", !isActive && "opacity-0")} aria-hidden />
        {label}
      </button>
    </th>
  );
}
