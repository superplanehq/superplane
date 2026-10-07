export function FactoryNodeStepList({ steps }: { steps: string[] }) {
  if (steps.length === 0) {
    return null;
  }

  return (
    <ol
      className="border-t border-border/70 bg-muted/30 px-3.5 py-2.5"
      data-testid="factory-node-step-list"
      aria-label="Steps"
    >
      {steps.map((step, index) => (
        <li
          key={`${index}-${step}`}
          className="flex min-h-7 items-center gap-2 border-b border-border/50 py-1.5 last:border-b-0"
        >
          <span
            className="flex size-4 shrink-0 items-center justify-center rounded-full bg-muted text-[9px] font-semibold tabular-nums text-muted-foreground"
            aria-hidden
          >
            {index + 1}
          </span>
          <span className="min-w-0 text-[12px] leading-4 font-medium text-card-foreground">{step}</span>
        </li>
      ))}
    </ol>
  );
}
