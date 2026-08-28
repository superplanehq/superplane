import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { FileText, Maximize2, Minimize2, UserPlus, XIcon } from "lucide-react";
import type { ReactNode } from "react";

import { FACTORIES_ORGANIZATION_ID } from "../../__fixtures__/factoryPageResponses";
import { ClickToRename } from "../../layout/ClickToRename";
import { extractArtifactMarkdownBody, toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { OrgUserReference } from "../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { WorkOrderAssigneesPopover } from "../../WorkOrderAssigneesPopover";
import { WorkOrderStatusNote } from "../../WorkOrderStatusNote";
import { RunOverlayBoardBackdrop, RunOverlayFrame } from "../work-order-run-overlay/runOverlayShared";
import type { PopupFixture, PopupLogEntry, PopupLogState } from "./workOrderPopupMocks";

export { RunOverlayBoardBackdrop };

export function PopupShell({
  testId,
  children,
  fixed = false,
  wide = false,
  canvas = false,
  fullPage = false,
  onDismiss,
}: {
  testId: string;
  children: ReactNode;
  fixed?: boolean;
  wide?: boolean;
  canvas?: boolean;
  fullPage?: boolean;
  onDismiss?: () => void;
}) {
  return (
    <RunOverlayFrame
      testId={testId}
      fixed={fixed}
      wide={wide}
      canvas={canvas}
      fullPage={fullPage}
      onDismiss={onDismiss}
    >
      {children}
    </RunOverlayFrame>
  );
}

const OPEN_FULL_SCREEN_LABEL = "Open full screen";
const EXIT_FULL_SCREEN_LABEL = "Exit full screen";
const POPUP_HEADER_ICON_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10";

function PopupFullScreenButton({ expanded, onToggle }: { expanded: boolean; onToggle: () => void }) {
  const label = expanded ? EXIT_FULL_SCREEN_LABEL : OPEN_FULL_SCREEN_LABEL;
  return (
    <button
      type="button"
      onClick={onToggle}
      className={POPUP_HEADER_ICON_BUTTON}
      aria-label={label}
      data-testid="popup-fullscreen-button"
    >
      {expanded ? <Minimize2 className="h-4 w-4" aria-hidden /> : <Maximize2 className="h-4 w-4" aria-hidden />}
    </button>
  );
}

export function PopupHeader({
  title,
  children,
  onClose,
  actions,
  expanded = false,
  onToggleExpanded,
  canEditTitle = false,
  titleBusy = false,
  onTitleSave,
  titleTestId = "popup-work-order-title",
  titleAriaLabel = "Work order title",
}: {
  title: string;
  children?: ReactNode;
  onClose?: () => void;
  actions?: ReactNode;
  expanded?: boolean;
  onToggleExpanded?: () => void;
  canEditTitle?: boolean;
  titleBusy?: boolean;
  onTitleSave?: (next: string) => void;
  titleTestId?: string;
  titleAriaLabel?: string;
}) {
  return (
    <header className="relative shrink-0 border-b border-border px-5 py-3">
      <div className="flex min-w-0 items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-3">
            <h2 className="min-w-0 flex-1 truncate text-[16px] font-semibold tracking-[-0.02em] text-foreground">
              {canEditTitle && onTitleSave ? (
                <ClickToRename
                  value={title}
                  onSave={onTitleSave}
                  canEdit={canEditTitle}
                  busy={titleBusy}
                  testId={titleTestId}
                  ariaLabel={titleAriaLabel}
                  className="max-w-full text-[16px] font-semibold tracking-[-0.02em]"
                  inputClassName="text-[16px] font-semibold tracking-[-0.02em]"
                />
              ) : (
                title
              )}
            </h2>
          </div>
          {children}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {actions}
          {onToggleExpanded ? <PopupFullScreenButton expanded={expanded} onToggle={onToggleExpanded} /> : null}
          <button type="button" onClick={onClose} className={POPUP_HEADER_ICON_BUTTON} aria-label="Close">
            <XIcon className="h-4 w-4" />
          </button>
        </div>
      </div>
    </header>
  );
}

type OwnerTimeCostFields = Pick<PopupFixture, "owner" | "costUsd" | "tokensLabel">;

/** Owner and spend. No elapsed time, status, author, or ticket key. */
export function OwnerTimeCostRow({
  fixture,
  className,
  children,
  organizationId,
  canEditOwner = false,
  assigneeIds = [],
  ownerBusy = false,
  onOwnerSave,
}: {
  fixture: OwnerTimeCostFields;
  className?: string;
  children?: ReactNode;
  organizationId?: string;
  canEditOwner?: boolean;
  assigneeIds?: string[];
  ownerBusy?: boolean;
  onOwnerSave?: (assigneeIds: string[]) => Promise<void>;
}) {
  const ownerMark = (
    <span className="inline-flex min-w-0 items-center gap-1.5">
      {assigneeIds.length > 0 || !canEditOwner ? (
        <OrgUserReference display={fixture.owner} size="xs" nameClassName="truncate text-[13px]" />
      ) : (
        <span className="inline-flex items-center gap-1 text-muted-foreground">
          <UserPlus className="size-3.5 shrink-0" aria-hidden />
          Assign
        </span>
      )}
    </span>
  );

  return (
    <div
      className={cn("mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px] text-foreground", className)}
      data-testid="popup-owner-time-cost"
    >
      {canEditOwner && organizationId && onOwnerSave ? (
        <WorkOrderAssigneesPopover
          organizationId={organizationId}
          selectedIds={assigneeIds}
          canEdit={canEditOwner}
          isSaving={ownerBusy}
          onSave={onOwnerSave}
          align="start"
        >
          <button
            type="button"
            className="inline-flex min-w-0 items-center gap-1.5 rounded-sm hover:bg-muted/60"
            aria-label={assigneeIds.length > 0 ? `Owner: ${fixture.owner.name}` : "Assign owner"}
            data-testid="popup-edit-owner"
            disabled={ownerBusy}
          >
            {ownerMark}
          </button>
        </WorkOrderAssigneesPopover>
      ) : (
        ownerMark
      )}
      <span className="text-foreground">
        {fixture.costUsd} <span className="text-muted-foreground">·</span> {fixture.tokensLabel}
      </span>
      {children}
    </div>
  );
}

export function DescriptionMarkdown({ artifact }: { artifact: FactoriesWorkOrderArtifact }) {
  const body = extractArtifactMarkdownBody(toArtifactDataRecord(artifact.data)) ?? "";
  return <MarkdownContent content={body} variant="workspace" />;
}

export function DescriptionFileCard({ artifact }: { artifact: FactoriesWorkOrderArtifact }) {
  return (
    <article className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex items-center gap-2 border-b border-border bg-muted/40 px-3 py-2">
        <FileText className="size-3.5 text-muted-foreground" aria-hidden />
        <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">description.md</span>
      </div>
      <div className="px-4 py-3">
        <DescriptionMarkdown artifact={artifact} />
      </div>
    </article>
  );
}

export function OutputList({ artifacts }: { artifacts: FactoriesWorkOrderArtifact[] }) {
  return (
    <ul className="flex flex-col">
      {artifacts.map((artifact) => (
        <li className="flex items-center py-1.5" key={artifact.id ?? `${artifact.type}-${artifact.createdAt}`}>
          <WorkOrderArtifactInline
            className="w-full justify-start"
            artifact={{
              id: artifact.id,
              type: artifact.type ?? "TYPE_UNSPECIFIED",
              data: toArtifactDataRecord(artifact.data),
            }}
          />
        </li>
      ))}
    </ul>
  );
}

export function WaitingNotes({ notes }: { notes: PopupFixture["waitingNotes"] }) {
  if (notes.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-3">
      {notes.map((note) => (
        <WorkOrderStatusNote
          key={note.key}
          note={note}
          organizationId={FACTORIES_ORGANIZATION_ID}
          canClose={false}
          canManage={false}
          isBusy={false}
          statusActions={[]}
          onClose={() => undefined}
          onStatusChange={async () => undefined}
        />
      ))}
    </div>
  );
}

const LOG_DOT: Record<PopupLogState, string> = {
  passed: "bg-[color:var(--status-completed-dot)]",
  running: "bg-[color:var(--status-running-dot)]",
  waiting: "bg-[color:var(--status-waiting-dot)]",
  failed: "bg-[color:var(--status-failed-dot)]",
};

export function AgentLogList({ entries }: { entries: PopupLogEntry[] }) {
  return (
    <ol className="flex flex-col">
      {entries.map((entry) => (
        <li key={entry.id} className="flex items-start gap-3 py-2">
          <span className={cn("mt-1.5 size-1.5 shrink-0 rounded-full", LOG_DOT[entry.state])} aria-hidden />
          <div className="min-w-0 flex-1">
            <p className="text-[13px] text-foreground">
              <span className="font-medium">{entry.title}</span>
              <span className="text-muted-foreground"> · {entry.actor}</span>
            </p>
            {entry.detail ? <p className="mt-0.5 text-[12px] text-muted-foreground">{entry.detail}</p> : null}
          </div>
          <span className="shrink-0 text-[12px] tabular-nums text-muted-foreground">{entry.duration}</span>
        </li>
      ))}
    </ol>
  );
}

export function PopupBody({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("min-h-0 flex-1 overflow-y-auto px-5 py-4", className)}>{children}</div>;
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return <h3 className="workspace-section-title">{children}</h3>;
}

export function PrototypeSwitcher({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (id: string) => void;
  options: { id: string; label: string; pattern: string }[];
}) {
  return (
    <div className="pointer-events-none absolute inset-x-0 top-0 z-20 flex justify-center p-3">
      <div className="pointer-events-auto flex items-center gap-2 rounded-full border border-border bg-background/95 px-2 py-1.5 shadow-sm">
        {options.map((entry) => (
          <Button
            key={entry.id}
            type="button"
            size="xs"
            variant={value === entry.id ? "default" : "ghost"}
            className={cn(value !== entry.id && "text-muted-foreground")}
            onClick={() => onChange(entry.id)}
            aria-pressed={value === entry.id}
          >
            {entry.label} · {entry.pattern}
          </Button>
        ))}
      </div>
    </div>
  );
}
