import { useEffect, useState, type FormEvent, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import { CopyLinkButton } from "../../CopyLinkButton";
import { getWorkOrderDisplayStatusMeta } from "../../lib/workOrderProgress";
import {
  primaryActionLabel,
  TASK_PAGE_COPY,
  type TaskPageActivityItem,
  type TaskPageCheck,
  type TaskPageRecord,
  type TaskPageRelatedItem,
  type TaskPageView,
} from "./taskPageModel";

interface TaskPageProps {
  view: TaskPageView;
  onTitleSave?: (title: string) => void;
  onPrimaryAction?: () => void;
  onRetry?: () => void;
  onCommentSubmit?: (body: string) => void;
}

export function TaskPage({ view, onTitleSave, onPrimaryAction, onRetry, onCommentSubmit }: TaskPageProps) {
  if (view.state === "loading") {
    return <TaskPageLoading />;
  }
  if (view.state === "error") {
    return <TaskPageError onRetry={onRetry} />;
  }
  return (
    <TaskPageRecordView
      record={view.record}
      onTitleSave={onTitleSave}
      onPrimaryAction={onPrimaryAction}
      onCommentSubmit={onCommentSubmit}
    />
  );
}

function TaskPageRecordView({
  record,
  onTitleSave,
  onPrimaryAction,
  onCommentSubmit,
}: {
  record: TaskPageRecord;
  onTitleSave?: (title: string) => void;
  onPrimaryAction?: () => void;
  onCommentSubmit?: (body: string) => void;
}) {
  return (
    <TaskPageSurface testId="task-page">
      <TaskPageHeader record={record} onTitleSave={onTitleSave} onPrimaryAction={onPrimaryAction} />
      <div className="mt-8 grid items-start gap-10 lg:grid-cols-[minmax(0,1fr)_17.5rem]">
        <TaskPageMain record={record} onCommentSubmit={onCommentSubmit} />
        <TaskPageProperties record={record} />
      </div>
    </TaskPageSurface>
  );
}

function TaskPageLoading() {
  return (
    <TaskPageSurface testId="task-page-loading">
      <p className="sr-only">{TASK_PAGE_COPY.loading}</p>
      <div className="h-3 w-16 animate-pulse rounded bg-black/8 dark:bg-white/10" />
      <div className="mt-3 h-8 w-3/4 max-w-xl animate-pulse rounded bg-black/8 dark:bg-white/10" />
      <div className="mt-8 grid items-start gap-10 lg:grid-cols-[minmax(0,1fr)_17.5rem]">
        <div className="space-y-3">
          <div className="h-3 w-24 animate-pulse rounded bg-black/8 dark:bg-white/10" />
          <div className="h-24 animate-pulse rounded bg-black/8 dark:bg-white/10" />
        </div>
        <div className="space-y-2">
          {Array.from({ length: 6 }, (_, index) => (
            <div key={index} className="h-8 animate-pulse rounded bg-black/8 dark:bg-white/10" />
          ))}
        </div>
      </div>
    </TaskPageSurface>
  );
}

function TaskPageError({ onRetry }: { onRetry?: () => void }) {
  return (
    <TaskPageSurface testId="task-page-error">
      <div className="flex max-w-lg flex-col gap-3 py-16">
        <h1 className="text-[22px] font-semibold tracking-[-0.03em] text-foreground">{TASK_PAGE_COPY.errorTitle}</h1>
        <p className="text-[13px] leading-5 text-muted-foreground">{TASK_PAGE_COPY.errorBody}</p>
        {onRetry ? (
          <Button type="button" size="sm" onClick={onRetry} className="mt-2 w-fit" data-testid="task-page-retry">
            {TASK_PAGE_COPY.retry}
          </Button>
        ) : null}
      </div>
    </TaskPageSurface>
  );
}

function TaskPageSurface({ children, testId }: { children: ReactNode; testId: string }) {
  return (
    <div className="min-h-full bg-[#f7f6f3] text-[13px] tracking-[-0.01em] dark:bg-background" data-testid={testId}>
      <div className="mx-auto w-full max-w-[70rem] px-8 py-8">{children}</div>
    </div>
  );
}

function TaskPageHeader({
  record,
  onTitleSave,
  onPrimaryAction,
}: {
  record: TaskPageRecord;
  onTitleSave?: (title: string) => void;
  onPrimaryAction?: () => void;
}) {
  return (
    <header className="flex flex-col gap-2">
      <p className="text-[11px] font-medium tracking-[0.02em] text-muted-foreground">{record.key}</p>
      <div className="flex items-start justify-between gap-4">
        <TaskTitle title={record.title} onSave={onTitleSave} />
        <div className="flex shrink-0 items-center gap-1.5 pt-1">
          <CopyLinkButton
            className="flex size-7 items-center justify-center rounded-full text-muted-foreground hover:bg-black/5 hover:text-foreground dark:hover:bg-white/10"
            iconClassName="size-3.5"
            testId="task-page-copy-link"
          />
          {record.primaryAction ? (
            <Button type="button" size="sm" onClick={onPrimaryAction} data-testid="task-page-primary-action">
              {primaryActionLabel(record.primaryAction)}
            </Button>
          ) : null}
        </div>
      </div>
    </header>
  );
}

function TaskTitle({ title, onSave }: { title: string; onSave?: (next: string) => void }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(title);

  useEffect(() => {
    if (!editing) {
      setDraft(title);
    }
  }, [editing, title]);

  const commit = () => {
    const next = draft.trim();
    if (next && next !== title) {
      onSave?.(next);
    }
    setEditing(false);
  };

  if (editing && onSave) {
    return (
      <div className="min-w-0">
        <Label htmlFor="task-page-title-input" className="sr-only">
          {TASK_PAGE_COPY.titleAriaLabel}
        </Label>
        <Input
          id="task-page-title-input"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              commit();
            }
            if (event.key === "Escape") {
              setDraft(title);
              setEditing(false);
            }
          }}
          data-testid="task-page-title-input"
          className="h-auto min-h-10 rounded-md border-border bg-background px-2 py-1 text-[26px] font-semibold leading-tight tracking-[-0.03em] shadow-none"
        />
      </div>
    );
  }

  return (
    <h1 className="min-w-0 text-[26px] font-semibold leading-tight tracking-[-0.03em] text-foreground">
      {onSave ? (
        <button
          type="button"
          onClick={() => setEditing(true)}
          className="rounded-sm text-left hover:bg-black/4 dark:hover:bg-white/8"
          data-testid="task-page-title"
          aria-label={TASK_PAGE_COPY.titleAriaLabel}
        >
          {title}
        </button>
      ) : (
        <span data-testid="task-page-title">{title}</span>
      )}
    </h1>
  );
}

function TaskPageMain({
  record,
  onCommentSubmit,
}: {
  record: TaskPageRecord;
  onCommentSubmit?: (body: string) => void;
}) {
  return (
    <div className="min-w-0 space-y-8">
      <TaskSection title={TASK_PAGE_COPY.writeUp} testId="task-page-write-up">
        {record.description ? (
          <MarkdownContent content={record.description} variant="workspace" />
        ) : (
          <EmptyLine>{TASK_PAGE_COPY.noWriteUp}</EmptyLine>
        )}
      </TaskSection>

      <TaskSection title={TASK_PAGE_COPY.checks} testId="task-page-checks">
        <CompactList
          items={record.checks}
          empty={TASK_PAGE_COPY.noChecks}
          renderItem={(check) => <CheckRow check={check} />}
        />
      </TaskSection>

      <TaskSection title={TASK_PAGE_COPY.artifacts} testId="task-page-artifacts">
        <RelatedList items={record.artifacts} empty={TASK_PAGE_COPY.noArtifacts} />
      </TaskSection>

      <TaskSection title={TASK_PAGE_COPY.pullRequests} testId="task-page-pull-requests">
        <RelatedList items={record.pullRequests} empty={TASK_PAGE_COPY.noPullRequests} />
      </TaskSection>

      <TaskSection title={TASK_PAGE_COPY.activity} testId="task-page-activity">
        <ActivityList items={record.activity} onCommentSubmit={onCommentSubmit} />
      </TaskSection>
    </div>
  );
}

function TaskPageProperties({ record }: { record: TaskPageRecord }) {
  const statusMeta = getWorkOrderDisplayStatusMeta(record.status);
  return (
    <dl className="min-w-0" data-testid="task-page-properties">
      <PropertyRow label={TASK_PAGE_COPY.status}>
        <span
          className={cn("inline-flex rounded-full border px-2 py-0.5 text-[12px] font-medium", statusMeta.className)}
        >
          {record.statusLabel}
        </span>
      </PropertyRow>
      <PropertyRow label={TASK_PAGE_COPY.author}>{record.author}</PropertyRow>
      <PropertyRow label={TASK_PAGE_COPY.assignees}>
        {record.assignees.length > 0 ? record.assignees.join(", ") : <Muted>{TASK_PAGE_COPY.noOwner}</Muted>}
      </PropertyRow>
      {record.mission ? <PropertyRow label={TASK_PAGE_COPY.mission}>{record.mission}</PropertyRow> : null}
      {record.createdLabel ? <PropertyRow label={TASK_PAGE_COPY.created}>{record.createdLabel}</PropertyRow> : null}
      <PropertyRow label={TASK_PAGE_COPY.spend}>
        {record.spendLabel ? record.spendLabel : <Muted>{TASK_PAGE_COPY.noSpend}</Muted>}
      </PropertyRow>
      <PropertyRow label={TASK_PAGE_COPY.source}>
        {record.source.href ? (
          <a href={record.source.href} target="_blank" rel="noopener noreferrer" className="truncate hover:underline">
            {record.source.label}
          </a>
        ) : (
          record.source.label
        )}
      </PropertyRow>
      <PropertyRow label={TASK_PAGE_COPY.factoryLines} last>
        {record.factoryLines.length > 0 ? record.factoryLines.join(", ") : <Muted>{TASK_PAGE_COPY.noLines}</Muted>}
      </PropertyRow>
    </dl>
  );
}

function PropertyRow({ label, children, last = false }: { label: string; children: ReactNode; last?: boolean }) {
  return (
    <div
      className={cn(
        "grid grid-cols-[7rem_minmax(0,1fr)] items-center gap-3 py-2.5",
        !last && "border-b border-black/8 dark:border-white/10",
      )}
    >
      <dt className="text-[12px] text-muted-foreground">{label}</dt>
      <dd className="min-w-0 truncate text-[13px] text-foreground">{children}</dd>
    </div>
  );
}

function TaskSection({ title, testId, children }: { title: string; testId: string; children: ReactNode }) {
  return (
    <section data-testid={testId}>
      <h2 className="workspace-section-label">{title}</h2>
      <div className="mt-2">{children}</div>
    </section>
  );
}

function CompactList<T extends { id: string }>({
  items,
  empty,
  renderItem,
}: {
  items: T[];
  empty: string;
  renderItem: (item: T) => ReactNode;
}) {
  if (items.length === 0) {
    return <EmptyLine>{empty}</EmptyLine>;
  }
  return (
    <ul className="divide-y divide-black/8 dark:divide-white/10">
      {items.map((item) => (
        <li key={item.id} className="py-2 first:pt-0 last:pb-0">
          {renderItem(item)}
        </li>
      ))}
    </ul>
  );
}

function CheckRow({ check }: { check: TaskPageCheck }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <div className="flex items-baseline gap-2">
        <span className="tabular-nums font-medium text-foreground">{check.scoreLabel}</span>
        <span className="text-muted-foreground">{check.name}</span>
      </div>
      {check.summary ? <p className="text-[12px] leading-5 text-muted-foreground">{check.summary}</p> : null}
    </div>
  );
}

function ActivityItem({ item }: { item: TaskPageActivityItem }) {
  if (item.kind === "comment") {
    return (
      <li className="min-w-0">
        <p className="text-[13px] leading-5">
          <span className="font-medium text-foreground">{item.actor}</span>{" "}
          <span className="text-muted-foreground">commented · {item.timeLabel}</span>
        </p>
        <p className="mt-1 text-[13px] leading-5 text-foreground">{item.text}</p>
      </li>
    );
  }

  return (
    <li className="min-w-0">
      <p className="text-[13px] leading-5">
        <span className="font-medium text-foreground">{item.actor}</span>{" "}
        <span className="text-muted-foreground">{item.text}</span>{" "}
        <span className="text-[12px] text-muted-foreground">· {item.timeLabel}</span>
      </p>
    </li>
  );
}

function RelatedList({ items, empty }: { items: TaskPageRelatedItem[]; empty: string }) {
  return (
    <CompactList
      items={items}
      empty={empty}
      renderItem={(item) =>
        item.href ? (
          <a href={item.href} target="_blank" rel="noopener noreferrer" className="truncate hover:underline">
            {item.title}
          </a>
        ) : (
          <span className="truncate">{item.title}</span>
        )
      }
    />
  );
}

function ActivityList({
  items,
  onCommentSubmit,
}: {
  items: TaskPageActivityItem[];
  onCommentSubmit?: (body: string) => void;
}) {
  return (
    <div className="space-y-4">
      {items.length === 0 ? (
        <EmptyLine>{TASK_PAGE_COPY.noActivity}</EmptyLine>
      ) : (
        <ul className="space-y-3">
          {items.map((item) => (
            <ActivityItem key={item.id} item={item} />
          ))}
        </ul>
      )}
      <CommentComposer onSubmit={onCommentSubmit} />
    </div>
  );
}

function CommentComposer({ onSubmit }: { onSubmit?: (body: string) => void }) {
  const [body, setBody] = useState("");
  const canSend = Boolean(body.trim());

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const next = body.trim();
    if (!next) {
      return;
    }
    onSubmit?.(next);
    setBody("");
  };

  return (
    <form className="space-y-2" onSubmit={handleSubmit} data-testid="task-page-comment-composer">
      <Label htmlFor="task-page-comment" className="sr-only">
        {TASK_PAGE_COPY.commentLabel}
      </Label>
      <Textarea
        id="task-page-comment"
        value={body}
        onChange={(event) => setBody(event.target.value)}
        placeholder={TASK_PAGE_COPY.commentPlaceholder}
        className="min-h-20 bg-white text-[13px] dark:bg-background"
      />
      <Button type="submit" size="sm" disabled={!canSend} data-testid="task-page-send-comment">
        {TASK_PAGE_COPY.sendComment}
      </Button>
    </form>
  );
}

function EmptyLine({ children }: { children: ReactNode }) {
  return <p className="text-[13px] text-muted-foreground">{children}</p>;
}

function Muted({ children }: { children: ReactNode }) {
  return <span className="text-muted-foreground">{children}</span>;
}
