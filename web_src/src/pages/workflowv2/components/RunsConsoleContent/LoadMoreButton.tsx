import { Loader2 } from "lucide-react";

export function LoadMoreButton({
  isFetchingNextPage,
  onLoadMore,
  loadedCount,
  totalCount,
}: {
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  loadedCount: number;
  totalCount: number;
}) {
  return (
    <div className="px-4 pt-2 pb-8 text-center">
      <button
        type="button"
        onClick={onLoadMore}
        disabled={isFetchingNextPage}
        className="text-xs font-medium text-slate-500 hover:text-slate-700 disabled:text-gray-400 transition-colors"
      >
        {isFetchingNextPage ? (
          <span className="inline-flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            Loading...
          </span>
        ) : (
          `Load more (${loadedCount} of ${totalCount})`
        )}
      </button>
    </div>
  );
}