export function DraftChangeDots({
  uncommitted,
  committed,
  testIdPrefix,
}: {
  uncommitted: boolean;
  committed: boolean;
  testIdPrefix: string;
}) {
  if (!uncommitted && !committed) {
    return null;
  }

  return (
    <span className="inline-flex items-center gap-0.5" aria-hidden="true">
      {committed ? (
        <span
          className="inline-flex size-1.5 shrink-0 rounded-full bg-blue-500"
          data-testid={`${testIdPrefix}-committed-dot`}
        />
      ) : null}
      {uncommitted ? (
        <span
          className="inline-flex size-1.5 shrink-0 rounded-full bg-orange-500"
          data-testid={`${testIdPrefix}-uncommitted-dot`}
        />
      ) : null}
    </span>
  );
}
