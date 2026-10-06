import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";

import { statusBadgeClass, visibleRange } from "./fleetAdmin";

export const RelativeTimestamp = ({ value }: { value?: string | null }) => (
  <Timestamp
    date={value ?? undefined}
    display="relative"
    className="text-gray-600 whitespace-nowrap dark:text-gray-400"
    fallback={<span className="text-gray-400 dark:text-gray-500">—</span>}
  />
);

export const StateBadge = ({ state }: { state: string }) => (
  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeClass(state, false)}`}>
    {state}
  </span>
);

const buttonClass =
  "rounded border border-slate-200 bg-white px-3 py-1 text-xs hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800";

export const PageControls = ({
  pageIndex,
  rowCount,
  totalCount,
  hasNextPage,
  onPrevious,
  onNext,
  previousTestId,
  nextTestId,
}: {
  pageIndex: number;
  rowCount: number;
  totalCount: number;
  hasNextPage: boolean;
  onPrevious: () => void;
  onNext: () => void;
  previousTestId: string;
  nextTestId: string;
}) => {
  const range = visibleRange(pageIndex, rowCount);
  const label = rowCount === 0 ? `Showing 0 of ${totalCount}` : `Showing ${range.start}–${range.end} of ${totalCount}`;

  return (
    <div className="mt-4 flex items-center justify-between text-sm text-gray-500 dark:text-gray-400">
      <Text className="text-sm text-gray-500 dark:text-gray-400">{label}</Text>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={onPrevious}
          disabled={pageIndex === 0}
          data-testid={previousTestId}
          className={buttonClass}
        >
          Previous
        </button>
        <button type="button" onClick={onNext} disabled={!hasNextPage} data-testid={nextTestId} className={buttonClass}>
          Next
        </button>
      </div>
    </div>
  );
};

export const tableClass =
  "overflow-hidden rounded-md bg-white shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70";

export const headerCellClass = "px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400";

export const rowClass =
  "border-b border-slate-50 last:border-0 hover:bg-slate-50 dark:border-gray-800/70 dark:hover:bg-gray-800/50";
