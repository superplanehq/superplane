import { ArrowDown, ArrowUp } from "lucide-react";

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

  return (
    <th
      className={`text-left px-4 py-2.5 text-gray-500 font-medium cursor-pointer select-none hover:text-gray-700 transition-colors ${className}`}
      onClick={() => onSort(field)}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {isActive && (currentDirection === "asc" ? <ArrowUp size={12} /> : <ArrowDown size={12} />)}
      </span>
    </th>
  );
}
