import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";

export function ReviewRequestDecisionSummary({
  run,
  executions,
}: {
  run: CanvasesCanvasRun;
  executions: CanvasesCanvasNodeExecution[];
}) {
  const scoped = executions.filter((execution) =>
    execution.runId ? execution.runId === run.id : run.executions?.some((ref) => ref.id === execution.id),
  );
  const decision = scoped
    .filter((execution) =>
      ["review-unnecessary", "stale-report", "no-reviewer", "review-outcome"].includes(execution.nodeId ?? ""),
    )
    .flatMap((execution) => execution.outputs?.default ?? [])
    .map((output) => record(record(record(output)?.data)?.reviewRequest))
    .find(Boolean);
  if (!decision || typeof decision.status !== "string" || typeof decision.rationale !== "string") return null;
  const labels: Record<string, string> = {
    review_unnecessary: "Review unnecessary",
    no_eligible_reviewer: "No eligible reviewer",
    requested: "Review requested",
    already_covered: "Review already covered",
    stale: "Stale report",
  };
  const reviewers = Array.isArray(decision.reviewers)
    ? decision.reviewers.flatMap((reviewer) => {
        const identity = record(reviewer);
        return typeof identity?.login === "string" ? [identity] : [];
      })
    : [];
  return (
    <section aria-label="Review decision" className="mb-6 space-y-3 rounded-lg border border-border bg-muted/30 p-5">
      <h2 className="text-sm font-semibold">{labels[decision.status] ?? decision.status}</h2>
      <p className="text-sm leading-6">{decision.rationale}</p>
      {decision.fallbackUsed === true ? <p className="text-sm text-muted-foreground">Fallback reviewer used.</p> : null}
      {reviewers.length ? (
        <ul className="space-y-2 text-sm">
          {reviewers.map((reviewer) => (
            <li key={String(reviewer.login)}>
              <span className="font-medium">{String(reviewer.login)}</span>
              {typeof reviewer.reason === "string" && reviewer.reason ? `: ${reviewer.reason}` : null}
            </li>
          ))}
        </ul>
      ) : decision.status === "no_eligible_reviewer" ? (
        <p className="text-sm text-muted-foreground">No reviewer was selected.</p>
      ) : null}
    </section>
  );
}

function record(value: unknown): Record<string, unknown> | undefined {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined;
}
