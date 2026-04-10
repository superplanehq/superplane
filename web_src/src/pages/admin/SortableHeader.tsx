import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";

export type SortDirection = "asc" | "desc";

interface SortableHeaderProps<TField extends string> {
  label: string;
  field: TField;
  currentSort: TField;
  currentDirection: SortDirection;
  onSort: (field: TField) => void;
  className?: string;
}

export function SortableHeader<TField extends string>({
  label,
  field,
  currentSort,
  currentDirection,
  onSort,
  className = "",
}: SortableHeaderProps<TField>) {
  const isActive = currentSort === field;
  const ariaSort = isActive ? (currentDirection === "asc" ? "ascending" : "descending") : "none";

  return (
    <th className={`px-4 py-2.5 ${className}`} aria-sort={ariaSort}>
      <button
        type="button"
        className="inline-flex w-full items-center gap-1 text-left text-gray-500 font-medium select-none hover:text-gray-700 transition-colors"
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
          <ArrowUpDown size={12} className="text-gray-300" />
        )}
      </button>
    </th>
  );
}
