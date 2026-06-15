import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import { cn } from "@/lib/utils";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import { useMemo } from "react";
import "react-diff-view/style/index.css";
import type { VersionNodeDiffItem, VersionNodeDiffSummary } from "./summarizeNodeDiff";

type DiffLine = VersionNodeDiffItem["lines"][number];

/** Shared +N added / ~M updated / -K removed row for version diff accordions. */
export function NodeDiffSummaryCounts({
  addedCount,
  updatedCount,
  removedCount,
  className,
}: {
  addedCount: number;
  updatedCount: number;
  removedCount: number;
  className?: string;
}) {
  return (
    <div
      className={cn("flex flex-wrap items-center gap-3 text-xs font-medium", className)}
      role="status"
      aria-live="polite"
    >
      <span className={cn(addedCount > 0 ? "text-emerald-600" : "text-slate-500")}>+{addedCount} added</span>
      <span className={cn(updatedCount > 0 ? "text-sky-600" : "text-slate-500")}>~{updatedCount} updated</span>
      <span className={cn(removedCount > 0 ? "text-red-600" : "text-slate-500")}>-{removedCount} removed</span>
    </div>
  );
}

function toUnifiedDiffText(lines: DiffLine[]): string {
  return lines
    .map((line) => {
      if (line.prefix === "meta") {
        return line.text;
      }

      if (line.prefix === "context") {
        if (line.text.startsWith("@@")) {
          return line.text;
        }

        return ` ${line.text}`;
      }

      return `${line.prefix}${line.text}`;
    })
    .join("\n");
}

function NodeGitDiff({ lines, nodeID }: { lines: DiffLine[]; nodeID: string }) {
  const files = useMemo(() => parseDiff(toUnifiedDiffText(lines), { nearbySequences: "zip" }), [lines]);

  if (files.length === 0) {
    return <p className="text-xs text-slate-600">No diff available for this node.</p>;
  }

  return (
    <div className="overflow-hidden rounded-md border border-slate-200 bg-white">
      <div className="max-h-96 overflow-auto">
        {files.map((file) => (
          <Diff
            key={`${nodeID}-${file.oldRevision}-${file.newRevision}`}
            viewType="split"
            diffType={file.type}
            hunks={file.hunks}
          >
            {(hunks) => hunks.map((hunk) => <Hunk key={`${nodeID}-${hunk.content}`} hunk={hunk} />)}
          </Diff>
        ))}
      </div>
    </div>
  );
}

export function VersionNodeDiffAccordion({
  summary,
  className,
  conflictingNodeIDs,
  emptyMessage = "No node changes found between these versions.",
}: {
  summary: VersionNodeDiffSummary;
  className?: string;
  conflictingNodeIDs?: Set<string>;
  emptyMessage?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <NodeDiffSummaryCounts
        addedCount={summary.addedCount}
        updatedCount={summary.updatedCount}
        removedCount={summary.removedCount}
      />
      {summary.items.length === 0 ? (
        <p className="text-xs text-slate-600">{emptyMessage}</p>
      ) : (
        <Accordion type="multiple" className="w-full rounded-md border border-slate-200 px-2">
          {summary.items.map((item, index) => (
            <AccordionItem
              key={`${item.id}-${item.changeType}-${index}`}
              value={`${item.id}-${item.changeType}-${index}`}
              className="border-b-0 border-slate-200"
            >
              <AccordionTrigger className="py-2 hover:no-underline">
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className={cn(
                      "inline-flex min-w-8 justify-center rounded px-1.5 py-0.5 text-[11px] font-semibold",
                      item.changeType === "removed"
                        ? "bg-red-100 text-red-700"
                        : item.changeType === "added"
                          ? "bg-emerald-100 text-emerald-700"
                          : "bg-sky-100 text-sky-700",
                    )}
                  >
                    {item.changeType === "updated" ? "+/-" : item.changeType === "removed" ? "-" : "+"}
                  </span>
                  <span className="truncate text-sm text-slate-900">{item.name}</span>
                  <span className="truncate text-xs text-slate-500">{item.id}</span>
                  {conflictingNodeIDs?.has(item.id) ? (
                    <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-red-700">
                      conflict
                    </span>
                  ) : null}
                </div>
              </AccordionTrigger>
              <AccordionContent>
                <NodeGitDiff lines={item.lines} nodeID={item.id} />
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      )}
    </div>
  );
}
