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
  const applied = scoped.find((execution) => execution.nodeId === "request-reviewers")?.metadata?.reviewRequest;
  const outputs = scoped.find((execution) => execution.nodeId === "select-reviewers")?.outputs?.passed;
  const output = Array.isArray(outputs)
    ? outputs.map((item) => record(record(record(item)?.data)?.reviewRequest)).find(Boolean)
    : undefined;
  const decision = record(applied) ?? output;
  if (!decision || typeof decision.reviewRequired !== "boolean" || typeof decision.rationale !== "string") return null;
  const reviewers = Array.isArray(decision.reviewers)
    ? decision.reviewers.flatMap((reviewer) => {
        const identity = record(reviewer);
        return typeof identity?.login === "string" ? [identity] : [];
      })
    : [];
  return (
    <section aria-label="Review decision" className="mb-6 space-y-3 rounded-lg border border-border bg-muted/30 p-5">
      <h2 className="text-sm font-semibold">{decision.reviewRequired ? "Review required" : "No review required"}</h2>
      <p className="text-sm leading-6">{decision.rationale}</p>
      {reviewers.length ? (
        <ul className="space-y-2 text-sm">
          {reviewers.map((reviewer) => (
            <li key={String(reviewer.login)}>
              <span className="font-medium">{String(reviewer.login)}</span>
              {typeof reviewer.reason === "string" && reviewer.reason ? `: ${reviewer.reason}` : null}
            </li>
          ))}
        </ul>
      ) : decision.reviewRequired ? (
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
