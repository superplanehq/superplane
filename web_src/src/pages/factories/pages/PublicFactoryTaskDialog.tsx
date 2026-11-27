import { useEffect, useState } from "react";
import { CheckCircle2, Circle, ExternalLink } from "lucide-react";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Link } from "@/components/Link/link";
import type { PublicBoard, PublicBoardCard } from "./publicBoardModel";

export interface PublicTask {
  card: PublicBoardCard;
  description: string;
  column: string;
  totalTokens: number;
  comments: { body: string; createdAt: string }[];
  runs: { name: string; status: string; result?: string; durationSeconds: number }[];
  pullRequests: { number: number; state: string; url: string }[];
}

export function PublicFactoryTaskDialog({
  url,
  workspaceKey,
  board,
  onClose,
}: {
  url: string;
  workspaceKey: string;
  board: PublicBoard;
  onClose: () => void;
}) {
  const [task, setTask] = useState<PublicTask | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    setTask(null);
    setError(false);
    void fetch(url, { credentials: "same-origin", cache: "no-store", signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error("Task unavailable");
        const next = (await response.json()) as PublicTask;
        if (!controller.signal.aborted) setTask(next);
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setTask(null);
          setError(true);
        }
      });
    return () => controller.abort();
  }, [url, board]);

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent size="large" className="flex max-h-[90vh] w-[94vw] max-w-6xl flex-col gap-0 overflow-hidden p-0">
        <header className="border-b px-6 py-5 pr-12">
          <DialogTitle className="leading-normal">
            {task ? (
              <>
                <span className="mr-3 font-normal text-muted-foreground">
                  {workspaceKey}-{task.card.number}
                </span>
                {task.card.title}
              </>
            ) : (
              "Task details"
            )}
          </DialogTitle>
          <DialogDescription className="mt-1">Public view</DialogDescription>
        </header>
        {task ? (
          <PublicTaskDetails task={task} />
        ) : (
          <p role="status" className="p-6 text-sm text-muted-foreground">
            {error ? "This task is not available." : "Loading task…"}
          </p>
        )}
      </DialogContent>
    </Dialog>
  );
}

// Public text is deliberately not rendered as rich media: embedded images,
// file previews and artifact references must not fetch private content.
function PublicTaskDetails({ task }: { task: PublicTask }) {
  return (
    <div className="grid min-h-0 gap-6 overflow-y-auto p-6 lg:grid-cols-[minmax(0,1fr)_320px]">
      <div className="min-w-0 space-y-6">
        <section aria-label="Intake">
          <h3 className="mb-3 text-sm font-semibold">Intake</h3>
          <div className="rounded-lg border bg-muted/20 p-4">
            <p className="whitespace-pre-wrap break-words text-sm leading-relaxed">
              {task.description || "No description."}
            </p>
          </div>
        </section>
        {task.runs.length > 0 ? (
          <section aria-label="Run summaries">
            <h3 className="mb-3 text-sm font-semibold">Activity</h3>
            <ol className="space-y-3">
              {task.runs.map((run, index) => (
                <li key={index} className="flex items-center gap-3 rounded-lg border bg-muted/20 p-4 text-sm">
                  {run.status === "finished" && run.result === "passed" ? (
                    <CheckCircle2 className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                  ) : (
                    <Circle className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                  )}
                  <span className="min-w-0 flex-1 break-words font-medium">{run.name || "Agent run"}</span>
                  <span className="text-muted-foreground">{run.result || run.status}</span>
                  <span className="text-muted-foreground">{Math.max(0, Math.round(run.durationSeconds))}s</span>
                </li>
              ))}
            </ol>
          </section>
        ) : null}
        <section aria-label="Comments">
          <h3 className="mb-3 text-sm font-semibold">Comments</h3>
          {task.comments.length === 0 ? (
            <p className="text-sm text-muted-foreground">No comments yet.</p>
          ) : (
            <ol className="space-y-3">
              {task.comments.map((comment, index) => (
                <li key={index} className="rounded-lg border p-4">
                  <time className="text-xs text-muted-foreground" dateTime={comment.createdAt}>
                    {new Date(comment.createdAt).toLocaleString()}
                  </time>
                  <p className="mt-2 whitespace-pre-wrap break-words text-sm">{comment.body}</p>
                </li>
              ))}
            </ol>
          )}
        </section>
      </div>
      <aside className="h-fit space-y-5 rounded-lg border p-4 text-sm" aria-label="Task overview">
        <h3 className="font-semibold">{task.column}</h3>
        {task.pullRequests.map((pr) => (
          <div key={`${pr.url}:${pr.number}`} className="flex items-center justify-between gap-2">
            {safePublicLink(pr.url) ? (
              <Link
                href={pr.url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 underline"
              >
                Review PR #{pr.number}
                <ExternalLink className="size-3" aria-hidden />
              </Link>
            ) : (
              <span>PR #{pr.number}</span>
            )}
            <span className="text-muted-foreground">{pr.state}</span>
          </div>
        ))}
        <dl className="space-y-3">
          {task.card.confidence != null ? (
            <div className="flex justify-between gap-3">
              <dt className="text-muted-foreground">Confidence</dt>
              <dd>{task.card.confidence}</dd>
            </div>
          ) : null}
          {task.card.assignee?.name ? (
            <div className="flex justify-between gap-3">
              <dt className="text-muted-foreground">Owner</dt>
              <dd>{task.card.assignee.name}</dd>
            </div>
          ) : null}
          <div className="flex justify-between gap-3">
            <dt className="text-muted-foreground">Created</dt>
            <dd>{new Date(task.card.createdAt).toLocaleDateString()}</dd>
          </div>
          <div className="flex justify-between gap-3">
            <dt className="text-muted-foreground">Tokens</dt>
            <dd>{task.totalTokens.toLocaleString()}</dd>
          </div>
        </dl>
      </aside>
    </div>
  );
}

function safePublicLink(value: string): boolean {
  try {
    const url = new URL(value);
    return ["https:", "http:"].includes(url.protocol) && !url.username && !url.password;
  } catch {
    return false;
  }
}
