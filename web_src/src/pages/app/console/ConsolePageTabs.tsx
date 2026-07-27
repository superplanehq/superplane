import { useEffect, useMemo, useRef, useState } from "react";
import { Check, Pencil, Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import type { ConsolePage } from "@/hooks/useCanvasData";

import { MAX_CONSOLE_PAGES } from "./consoleYaml";

/**
 * Horizontal tab strip rendered at the top of the console overlay. In
 * view mode the strip stays hidden until the console has at least two
 * pages, so today's single-page apps look identical. Edit mode always
 * shows the strip so `+ add page` stays discoverable and the whole
 * page-management surface (rename / reorder / remove) is reachable.
 *
 * The visual pattern deliberately mimics browser folder tabs to signal
 * "these are peer views of the same console" rather than a navigation
 * step — the header pills (Canvas / Console / Memory / Files) stay
 * untouched.
 */
export type ConsolePageTabsProps = {
  pages: ConsolePage[];
  activePageId: string | null;
  readOnly: boolean;
  onSelectPage: (pageId: string) => void;
  onAddPage: () => void;
  onRenamePage: (pageId: string, name: string) => void;
  onReorderPages: (fromIndex: number, toIndex: number) => void;
  onRemovePage: (pageId: string) => void;
};

export function ConsolePageTabs({
  pages,
  activePageId,
  readOnly,
  onSelectPage,
  onAddPage,
  onRenamePage,
  onReorderPages,
  onRemovePage,
}: ConsolePageTabsProps) {
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [renameDraft, setRenameDraft] = useState("");
  const renameInputRef = useRef<HTMLInputElement | null>(null);
  const [pendingRemoveId, setPendingRemoveId] = useState<string | null>(null);

  useEffect(() => {
    if (renamingId && renameInputRef.current) {
      renameInputRef.current.focus();
      renameInputRef.current.select();
    }
  }, [renamingId]);

  const pendingRemovePage = useMemo(
    () => pages.find((page) => page.id === pendingRemoveId) ?? null,
    [pages, pendingRemoveId],
  );

  // View mode with 0 or 1 pages has no controls worth showing; skip the
  // strip entirely so the console header looks identical to today for
  // existing single-page apps. Edit mode always shows the strip so the
  // "+ add page" control stays reachable (including on an empty console,
  // where it is the only way to create the first named page from the tab
  // strip itself).
  if (readOnly && pages.length <= 1) return null;

  const canAddPage = !readOnly && pages.length < MAX_CONSOLE_PAGES;
  const canRemovePage = !readOnly && pages.length > 1;

  const startRename = (page: ConsolePage) => {
    if (readOnly) return;
    setRenamingId(page.id);
    setRenameDraft(page.name || page.id);
  };

  const commitRename = () => {
    if (!renamingId) return;
    const nextName = renameDraft.trim();
    if (nextName) onRenamePage(renamingId, nextName);
    setRenamingId(null);
    setRenameDraft("");
  };

  const cancelRename = () => {
    setRenamingId(null);
    setRenameDraft("");
  };

  const confirmRemove = () => {
    if (pendingRemovePage) onRemovePage(pendingRemovePage.id);
    setPendingRemoveId(null);
  };

  return (
    <>
      <div
        className="flex h-9 shrink-0 items-center gap-1 overflow-x-auto border-b border-slate-200 bg-white px-3 dark:border-gray-700 dark:bg-gray-900"
        data-testid="console-page-tabs"
      >
        {pages.map((page, index) => (
          <PageTabItem
            key={page.id}
            page={page}
            index={index}
            active={page.id === activePageId}
            isRenaming={renamingId === page.id}
            readOnly={readOnly}
            canRemovePage={canRemovePage}
            renameDraft={renameDraft}
            renameInputRef={renameInputRef}
            onRenameDraftChange={setRenameDraft}
            onCommitRename={commitRename}
            onCancelRename={cancelRename}
            onStartRename={startRename}
            onSelectPage={onSelectPage}
            onReorderPages={onReorderPages}
            onRequestRemove={setPendingRemoveId}
          />
        ))}
        {!readOnly ? (
          <button
            type="button"
            onClick={onAddPage}
            disabled={!canAddPage}
            className={cn(
              "ml-1 flex h-7 items-center justify-center rounded-md border border-dashed px-2 text-xs",
              "text-slate-600 hover:bg-slate-100 dark:text-gray-400 dark:hover:bg-gray-800",
              !canAddPage && "cursor-not-allowed opacity-50",
            )}
            title={canAddPage ? "Add page" : `Max ${MAX_CONSOLE_PAGES} pages per console`}
            data-testid="console-page-tabs-add"
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
        ) : null}
      </div>

      <RemovePageDialog page={pendingRemovePage} onCancel={() => setPendingRemoveId(null)} onConfirm={confirmRemove} />
    </>
  );
}

type PageTabItemProps = {
  page: ConsolePage;
  index: number;
  active: boolean;
  isRenaming: boolean;
  readOnly: boolean;
  canRemovePage: boolean;
  renameDraft: string;
  renameInputRef: React.RefObject<HTMLInputElement | null>;
  onRenameDraftChange: (next: string) => void;
  onCommitRename: () => void;
  onCancelRename: () => void;
  onStartRename: (page: ConsolePage) => void;
  onSelectPage: (pageId: string) => void;
  onReorderPages: (fromIndex: number, toIndex: number) => void;
  onRequestRemove: (pageId: string) => void;
};

function PageTabItem({
  page,
  index,
  active,
  isRenaming,
  readOnly,
  canRemovePage,
  renameDraft,
  renameInputRef,
  onRenameDraftChange,
  onCommitRename,
  onCancelRename,
  onStartRename,
  onSelectPage,
  onReorderPages,
  onRequestRemove,
}: PageTabItemProps) {
  return (
    <div
      className={cn(
        "group flex h-7 items-center gap-1 rounded-md border px-2 text-xs transition-colors",
        active
          ? "border-slate-300 bg-slate-100 text-slate-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
          : "border-transparent text-slate-600 hover:bg-slate-50 dark:text-gray-400 dark:hover:bg-gray-800/60",
      )}
      draggable={!readOnly && !isRenaming}
      onDragStart={(event) => event.dataTransfer.setData("text/plain", String(index))}
      onDragOver={(event) => {
        if (readOnly) return;
        event.preventDefault();
      }}
      onDrop={(event) => {
        if (readOnly) return;
        event.preventDefault();
        const from = Number(event.dataTransfer.getData("text/plain"));
        if (Number.isNaN(from) || from === index) return;
        onReorderPages(from, index);
      }}
      data-testid={`console-page-tab-${page.id}`}
      data-active={active ? "true" : undefined}
    >
      {isRenaming ? (
        <PageTabRenameField
          page={page}
          value={renameDraft}
          inputRef={renameInputRef}
          onChange={onRenameDraftChange}
          onCommit={onCommitRename}
          onCancel={onCancelRename}
        />
      ) : (
        <PageTabDisplay
          page={page}
          active={active}
          readOnly={readOnly}
          canRemovePage={canRemovePage}
          onSelectPage={onSelectPage}
          onStartRename={onStartRename}
          onRequestRemove={onRequestRemove}
        />
      )}
    </div>
  );
}

type PageTabRenameFieldProps = {
  page: ConsolePage;
  value: string;
  inputRef: React.RefObject<HTMLInputElement | null>;
  onChange: (next: string) => void;
  onCommit: () => void;
  onCancel: () => void;
};

function PageTabRenameField({ page, value, inputRef, onChange, onCommit, onCancel }: PageTabRenameFieldProps) {
  return (
    <div className="flex items-center gap-1">
      <input
        ref={inputRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            onCommit();
          }
          if (e.key === "Escape") {
            e.preventDefault();
            onCancel();
          }
        }}
        onBlur={onCommit}
        className="h-5 w-32 rounded border border-slate-300 bg-white px-1 text-xs text-slate-900 outline-none dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
        data-testid={`console-page-tab-${page.id}-rename-input`}
      />
      <button
        type="button"
        className="text-slate-500 hover:text-slate-800 dark:text-gray-400 dark:hover:text-gray-100"
        onMouseDown={(e) => e.preventDefault()}
        onClick={onCommit}
        aria-label="Confirm rename"
      >
        <Check className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

type PageTabDisplayProps = {
  page: ConsolePage;
  active: boolean;
  readOnly: boolean;
  canRemovePage: boolean;
  onSelectPage: (pageId: string) => void;
  onStartRename: (page: ConsolePage) => void;
  onRequestRemove: (pageId: string) => void;
};

function PageTabDisplay({
  page,
  active,
  readOnly,
  canRemovePage,
  onSelectPage,
  onStartRename,
  onRequestRemove,
}: PageTabDisplayProps) {
  return (
    <>
      <button
        type="button"
        onClick={() => onSelectPage(page.id)}
        onDoubleClick={() => onStartRename(page)}
        className="max-w-[16rem] truncate outline-none"
        data-testid={`console-page-tab-${page.id}-label`}
      >
        {page.name || page.id}
      </button>
      {!readOnly ? (
        <div className={cn("flex items-center gap-0.5", active ? "opacity-100" : "opacity-0 group-hover:opacity-100")}>
          <button
            type="button"
            onClick={() => onStartRename(page)}
            aria-label={`Rename page ${page.name || page.id}`}
            className="rounded p-0.5 text-slate-500 hover:bg-slate-200 hover:text-slate-800 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-100"
            data-testid={`console-page-tab-${page.id}-rename`}
          >
            <Pencil className="h-3 w-3" />
          </button>
          {canRemovePage ? (
            <button
              type="button"
              onClick={() => onRequestRemove(page.id)}
              aria-label={`Remove page ${page.name || page.id}`}
              className="rounded p-0.5 text-slate-500 hover:bg-red-100 hover:text-red-700 dark:text-gray-400 dark:hover:bg-red-950/50 dark:hover:text-red-400"
              data-testid={`console-page-tab-${page.id}-remove`}
            >
              <X className="h-3 w-3" />
            </button>
          ) : null}
        </div>
      ) : null}
    </>
  );
}

type RemovePageDialogProps = {
  page: ConsolePage | null;
  onCancel: () => void;
  onConfirm: () => void;
};

function RemovePageDialog({ page, onCancel, onConfirm }: RemovePageDialogProps) {
  return (
    <Dialog open={page !== null} onOpenChange={(open) => (open ? undefined : onCancel())}>
      <DialogContent className="dark:border-gray-600 dark:bg-gray-900">
        <DialogHeader>
          <DialogTitle>Remove page?</DialogTitle>
          <DialogDescription>
            {page ? (
              <>
                Removing <strong>{page.name || page.id}</strong> also deletes every panel on this page. This action
                cannot be undone.
              </>
            ) : (
              "Removing this page also deletes every panel on it."
            )}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} data-testid="console-page-remove-confirm">
            Remove page
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
