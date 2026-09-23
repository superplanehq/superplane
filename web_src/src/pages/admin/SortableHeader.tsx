import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";

export type SortDirection = "asc" | "desc";

interface SortableHeaderProps<TField extends string> {
  label: string;
  field: TField;
  currentSort: TField;
  currentDirection: SortDirection;
  onSort: (field: TField) => void;
  align?: "left" | "right";
  className?: string;
}

export function SortableHeader<TField extends string>({
  label,
  field,
  currentSort,
  currentDirection,
  onSort,
  align = "left",
  className = "",
}: SortableHeaderProps<TField>) {
  const isActive = currentSort === field;
  const ariaSort = isActive ? (currentDirection === "asc" ? "ascending" : "descending") : "none";
  const alignClass = align === "right" ? "justify-end text-right" : "justify-start text-left";

  return (
    <th className={`px-4 py-2.5 ${className}`} aria-sort={ariaSort}>
      <button
        type="button"
        className={`inline-flex w-full items-center gap-1 font-medium text-gray-500 select-none transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 ${alignClass}`}
        onClick={() => onSort(field)}
      >
        <span>{label}</span>
        {isActive ? (
          currentDirection === "asc" ? (
            <ArrowUp size={12} />
          ) : (
            <ArrowDown size={12} />
          )
        ) : (
          <ArrowUpDown size={12} className="text-gray-300 dark:text-gray-600" />
        )}
      </button>
    </th>
  );
}
