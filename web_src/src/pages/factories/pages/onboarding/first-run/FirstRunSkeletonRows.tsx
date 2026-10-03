/** Button-shaped placeholders for GitHub results that are still syncing. */
export function FirstRunSkeletonRows({ count, label, testId }: { count: number; label: string; testId: string }) {
  return (
    <div className="space-y-3" role="status" aria-label={label} data-testid={testId}>
      {Array.from({ length: count }, (_, index) => (
        <div key={index} className="h-9 w-full rounded-md bg-muted motion-safe:animate-pulse" />
      ))}
    </div>
  );
}
