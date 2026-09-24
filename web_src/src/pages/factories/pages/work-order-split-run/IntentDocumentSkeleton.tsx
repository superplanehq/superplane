import { Skeleton } from "@/ui/skeleton";

export const INTENT_DOCUMENT_LOADING_LABEL = "Loading the spec";

export function IntentDocumentSkeleton() {
  return (
    <div className="space-y-3" role="status" aria-label={INTENT_DOCUMENT_LOADING_LABEL} aria-busy="true">
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-11/12" />
      <Skeleton className="h-4 w-4/5" />
      <div className="h-6" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-5/6" />
      <Skeleton className="h-4 w-2/3" />
    </div>
  );
}
