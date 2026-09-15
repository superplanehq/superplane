import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import { Sparkline } from "@/pages/app/console/widget/Sparkline";
import type { RecurringPattern } from "@/pages/factories/verification/types";

interface RecurringPatternCardProps {
  pattern: RecurringPattern;
  onViewSuggestions: (patternId: string) => void;
}

/**
 * One recurring finding pattern: what it is, where it concentrates, the
 * standard fix, and how its occurrence count trends.
 */
export function RecurringPatternCard({ pattern, onViewSuggestions }: RecurringPatternCardProps) {
  const rising =
    pattern.occurrenceSeries.length > 1 &&
    pattern.occurrenceSeries[pattern.occurrenceSeries.length - 1] > pattern.occurrenceSeries[0];

  return (
    <article className={cn(factoryCardClassName, "flex flex-col gap-3 p-4")} aria-label={pattern.name}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-[13px] font-medium text-foreground">{pattern.name}</span>
          <p className="text-[13px] text-muted-foreground">{pattern.description}</p>
        </div>
        <div className="flex flex-col items-end gap-1">
          <span className="text-2xl font-medium tabular-nums text-slate-900 dark:text-gray-100">
            {pattern.openCount}
          </span>
          <span className="text-[11px] uppercase tracking-wide text-muted-foreground">Open</span>
        </div>
      </div>

      {pattern.occurrenceSeries.length > 1 ? (
        <Sparkline
          values={pattern.occurrenceSeries}
          className={rising ? "text-red-500 dark:text-red-400" : "text-emerald-500 dark:text-emerald-400"}
        />
      ) : null}

      <div className="rounded-md border border-border bg-background px-3 py-2">
        <p className="workspace-section-label text-muted-foreground">Top files</p>
        <ul className="mt-1 flex flex-col gap-1">
          {pattern.topFiles.map((file) => (
            <li key={file.path} className="flex items-center justify-between gap-2 text-[12px]">
              <code className="truncate font-mono text-slate-700 dark:text-gray-300">{file.path}</code>
              <span className="shrink-0 tabular-nums text-muted-foreground">{file.count}</span>
            </li>
          ))}
        </ul>
      </div>

      <p className="text-[13px] text-muted-foreground">
        <span className="font-medium text-foreground">Standard fix: </span>
        {pattern.remediation}
      </p>

      <div className="border-t border-border pt-3">
        <Button variant="outline" size="sm" onClick={() => onViewSuggestions(pattern.id)}>
          View matching suggestions
        </Button>
      </div>
    </article>
  );
}
