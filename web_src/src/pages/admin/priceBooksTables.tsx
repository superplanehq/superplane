import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { BookOpen } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import AdminPagination from "./AdminPagination";
import { SortableHeader, type SortDirection } from "./SortableHeader";
import { formatCentsPerMillionUsd, formatMicrosPerSecondUsdPerMinute } from "./priceBookFormat";
import type { PriceBookModelRate, PriceBookVMRate } from "./priceBooksApi";

export const tableWrapClass =
  "bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70";
const headerCellClass = "text-xs uppercase tracking-wide";
const numericHeaderCellClass = `${headerCellClass} text-right`;
const bodyCellClass = "px-4 py-2.5 text-gray-700 dark:text-gray-300";
const numericCellClass = `${bodyCellClass} text-right tabular-nums`;
const rowClass = "border-b border-slate-50 last:border-0 dark:border-gray-800/70";

export function EmptyRatesMessage({ message, action }: { message: string; action?: ReactNode }) {
  return (
    <div className={`${tableWrapClass} p-8 text-center`}>
      <BookOpen size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
      <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">{message}</Text>
      {action}
    </div>
  );
}

type IndexedModelRate = { rate: PriceBookModelRate; index: number };
type ModelSortField = "model" | "input" | "output" | "cache_read" | "cache_write" | "reasoning";

const MODEL_COLUMNS: { field: ModelSortField; label: string; numeric: boolean }[] = [
  { field: "model", label: "Model", numeric: false },
  { field: "input", label: "Input", numeric: true },
  { field: "output", label: "Output", numeric: true },
  { field: "cache_read", label: "Cache read", numeric: true },
  { field: "cache_write", label: "Cache write", numeric: true },
  { field: "reasoning", label: "Reasoning", numeric: true },
];

export function ModelsTable({ rows, pageSize }: { rows: IndexedModelRate[]; pageSize?: number }) {
  const sort = useColumnSort<ModelSortField>("model");
  const sorted = useMemo(() => sortModelRows(rows, sort.field, sort.direction), [rows, sort.direction, sort.field]);
  const offset = usePagedOffset(sort.field, sort.direction, pageSize);
  const visible = pageSize ? sorted.slice(offset.value, offset.value + pageSize) : sorted;

  if (rows.length === 0) {
    return null;
  }

  return (
    <>
      <div className={`${tableWrapClass} overflow-x-auto`}>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-100 dark:border-gray-700/70">
              {MODEL_COLUMNS.map((column) => (
                <SortableHeader
                  key={column.field}
                  label={column.label}
                  field={column.field}
                  currentSort={sort.field}
                  currentDirection={sort.direction}
                  onSort={sort.onSort}
                  align={column.numeric ? "right" : "left"}
                  className={column.numeric ? numericHeaderCellClass : headerCellClass}
                />
              ))}
            </tr>
          </thead>
          <tbody>
            {visible.map(({ rate }) => (
              <tr key={`${rate.provider}:${rate.match_key}:${rate.match_mode}`} className={rowClass}>
                <td className={`${bodyCellClass} font-mono text-xs`}>{rate.match_key}</td>
                <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.input_cents_per_million)}</td>
                <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.output_cents_per_million)}</td>
                <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.cache_read_cents_per_million)}</td>
                <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.cache_write_cents_per_million)}</td>
                <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.reasoning_cents_per_million)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {pageSize ? (
        <AdminPagination
          offset={offset.value}
          total={sorted.length}
          pageSize={pageSize}
          onPageChange={offset.setValue}
        />
      ) : null}
    </>
  );
}

type IndexedVMRate = { rate: PriceBookVMRate; index: number };
type VMSortField = "machine" | "micros" | "rate";

const VM_COLUMNS: { field: VMSortField; label: string; numeric: boolean }[] = [
  { field: "machine", label: "Machine type", numeric: false },
  { field: "micros", label: "Micros per second", numeric: true },
  { field: "rate", label: "Rate", numeric: true },
];

export function VMsTable({
  rates,
  canRemove,
  onRemove,
}: {
  rates: PriceBookVMRate[];
  canRemove: boolean;
  onRemove: (index: number) => void;
}) {
  const sort = useColumnSort<VMSortField>("machine");
  const sorted = useMemo(() => sortVMRows(rates, sort.field, sort.direction), [rates, sort.direction, sort.field]);

  if (rates.length === 0) {
    return <EmptyRatesMessage message="This version has no machine rates." />;
  }

  return (
    <div className={tableWrapClass}>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            {VM_COLUMNS.map((column) => (
              <SortableHeader
                key={column.field}
                label={column.label}
                field={column.field}
                currentSort={sort.field}
                currentDirection={sort.direction}
                onSort={sort.onSort}
                align={column.numeric ? "right" : "left"}
                className={column.numeric ? numericHeaderCellClass : headerCellClass}
              />
            ))}
            {canRemove && <th className="px-4 py-2.5" />}
          </tr>
        </thead>
        <tbody>
          {sorted.map(({ rate, index }) => (
            <tr key={`${rate.match_key}:${rate.match_mode}`} className={rowClass}>
              <td className={`${bodyCellClass} font-mono text-xs`}>{rate.match_key}</td>
              <td className={`${numericCellClass} font-mono text-xs`}>{rate.micros_per_second}</td>
              <td className={numericCellClass}>{formatMicrosPerSecondUsdPerMinute(rate.micros_per_second)}</td>
              {canRemove && (
                <td className={bodyCellClass}>
                  <Button type="button" variant="ghost" size="sm" onClick={() => onRemove(index)}>
                    Remove
                  </Button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function useColumnSort<TField extends string>(initialField: TField) {
  const [field, setField] = useState(initialField);
  const [direction, setDirection] = useState<SortDirection>("asc");

  const onSort = (next: TField) => {
    if (next === field) {
      setDirection((current) => (current === "asc" ? "desc" : "asc"));
      return;
    }
    setField(next);
    setDirection("asc");
  };

  return { field, direction, onSort };
}

function usePagedOffset(field: string, direction: SortDirection, pageSize: number | undefined) {
  const resetKey = pageSize ? `${field}:${direction}` : "";
  const [offset, setOffset] = useState(0);
  const [appliedResetKey, setAppliedResetKey] = useState(resetKey);
  if (appliedResetKey !== resetKey) {
    setAppliedResetKey(resetKey);
    setOffset(0);
  }
  return { value: offset, setValue: setOffset };
}

function sortModelRows(rows: IndexedModelRate[], field: ModelSortField, direction: SortDirection) {
  return [...rows].sort((left, right) => {
    const primary = compareModelField(left.rate, right.rate, field, direction);
    if (primary !== 0) {
      return primary;
    }
    return compareText(left.rate.match_key, right.rate.match_key, "asc");
  });
}

function compareModelField(
  left: PriceBookModelRate,
  right: PriceBookModelRate,
  field: ModelSortField,
  direction: SortDirection,
) {
  if (field === "model") {
    return compareText(left.match_key, right.match_key, direction);
  }
  return compareNumber(modelCents(left, field), modelCents(right, field), direction);
}

function modelCents(rate: PriceBookModelRate, field: Exclude<ModelSortField, "model">) {
  switch (field) {
    case "input":
      return rate.input_cents_per_million;
    case "output":
      return rate.output_cents_per_million;
    case "cache_read":
      return rate.cache_read_cents_per_million;
    case "cache_write":
      return rate.cache_write_cents_per_million;
    case "reasoning":
      return rate.reasoning_cents_per_million;
  }
}

function sortVMRows(rates: PriceBookVMRate[], field: VMSortField, direction: SortDirection) {
  const indexed: IndexedVMRate[] = rates.map((rate, index) => ({ rate, index }));
  return indexed.sort((left, right) => {
    const primary = compareVMField(left.rate, right.rate, field, direction);
    if (primary !== 0) {
      return primary;
    }
    return compareText(left.rate.match_key, right.rate.match_key, "asc");
  });
}

function compareVMField(left: PriceBookVMRate, right: PriceBookVMRate, field: VMSortField, direction: SortDirection) {
  if (field === "machine") {
    return compareText(left.match_key, right.match_key, direction);
  }
  return compareNumber(left.micros_per_second, right.micros_per_second, direction);
}

function compareText(left: string, right: string, direction: SortDirection) {
  const result = left.localeCompare(right);
  return direction === "asc" ? result : -result;
}

function compareNumber(left: number, right: number, direction: SortDirection) {
  const result = left - right;
  return direction === "asc" ? result : -result;
}
