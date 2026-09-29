import { Skeleton } from "@/ui/skeleton";

export function WorkOrderListSkeleton({ label }: { label: string }) {
  return (
    <div className="space-y-2 py-1" role="status" aria-label={label} aria-busy="true">
      <Skeleton className="h-4 w-5/6" />
      <Skeleton className="h-4 w-3/5" />
    </div>
  );
}
