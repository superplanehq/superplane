/** Manual-run target: node and template on one line. */
export function StartRunSummaryTable({
  nodeName,
  templateName,
}: {
  nodeName: string;
  templateName: string;
}) {
  return (
    <p className="text-xs text-slate-700" data-testid="start-run-summary">
      <span className="font-medium text-slate-800">{nodeName}</span>
      <span className="mx-1 text-slate-400">/</span>
      <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[11px] text-slate-700">{templateName}</code>
    </p>
  );
}
