import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { formatTimeAgo } from "@/lib/date";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { ExternalLink, FileText, GitPullRequest } from "lucide-react";
import { useState } from "react";

import {
  extractArtifactMarkdownBody,
  extractArtifactTitle,
  extractArtifactUrl,
  formatPrArtifactLabel,
} from "./lib/workOrderArtifact";
import { WorkOrderMarkdownArtifactDialog } from "./WorkOrderMarkdownArtifactDialog";

interface WorkOrderArtifactsPanelProps {
  artifacts: FactoriesWorkOrderArtifact[];
  isLoading: boolean;
  error?: Error | null;
}

export function WorkOrderArtifactsPanel({ artifacts, isLoading, error }: WorkOrderArtifactsPanelProps) {
  return (
    <section>
      <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Artifacts</h2>

      <div className="mt-3">
        {error ? (
          <p className="text-sm text-red-500 dark:text-red-400">Failed to load artifacts.</p>
        ) : isLoading ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading artifacts…</p>
        ) : artifacts.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            No artifacts yet. Automation nodes will attach PRs and notes here as they run.
          </p>
        ) : (
          <ul className="space-y-2">
            {artifacts.map((artifact) => (
              <WorkOrderArtifactRow key={artifact.id ?? `${artifact.type}-${artifact.createdAt}`} artifact={artifact} />
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}

function WorkOrderArtifactRow({ artifact }: { artifact: FactoriesWorkOrderArtifact }) {
  const isPr = artifact.type === "TYPE_PR";
  const data = toDataRecord(artifact.data);
  const title = extractArtifactTitle(data);

  return (
    <li className="rounded-md border border-gray-200 bg-white px-3 py-2 text-sm dark:border-gray-700/70 dark:bg-gray-900/40">
      {isPr ? <PrArtifactRow data={data} title={title} /> : <MarkdownArtifactRow data={data} title={title} />}

      <p className="mt-1 text-xs text-gray-500 dark:text-gray-500">
        {artifact.createdBy?.name ? `${artifact.createdBy.name} · ` : ""}
        {artifact.createdAt ? formatTimeAgo(new Date(artifact.createdAt)) : ""}
      </p>
    </li>
  );
}

function PrArtifactRow({ data, title }: { data: Record<string, unknown> | undefined; title: string | undefined }) {
  const safeUrl = safeExternalUrl(extractArtifactUrl(data));
  const prLabel = formatPrArtifactLabel(data);
  const label = prLabel || title || safeUrl || "Pull request";

  return (
    <div className="flex items-center gap-2">
      <GitPullRequest className="h-4 w-4 text-violet-500" aria-hidden />
      {safeUrl ? (
        <a
          href={safeUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex min-w-0 items-center gap-1 truncate font-medium text-violet-700 hover:underline dark:text-violet-300"
        >
          <span className="truncate">{label}</span>
          <ExternalLink className="h-3 w-3 shrink-0" aria-hidden />
        </a>
      ) : (
        <span className="min-w-0 truncate font-medium text-gray-900 dark:text-gray-100">{label}</span>
      )}
    </div>
  );
}

function MarkdownArtifactRow({
  data,
  title,
}: {
  data: Record<string, unknown> | undefined;
  title: string | undefined;
}) {
  const [open, setOpen] = useState(false);
  const body = extractArtifactMarkdownBody(data) ?? "";
  const label = title || "Note";

  return (
    <div className="flex items-center gap-2">
      <FileText className="h-4 w-4 text-emerald-500" aria-hidden />
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex min-w-0 items-center gap-1 truncate text-left font-medium text-violet-700 hover:underline dark:text-violet-300"
      >
        <span className="truncate">{label}</span>
      </button>
      <WorkOrderMarkdownArtifactDialog open={open} onClose={() => setOpen(false)} title={label} body={body} />
    </div>
  );
}

function toDataRecord(data: unknown): Record<string, unknown> | undefined {
  return data && typeof data === "object" ? (data as Record<string, unknown>) : undefined;
}
