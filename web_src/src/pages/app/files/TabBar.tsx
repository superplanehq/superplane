import { X } from "lucide-react";

import type { PendingFileChange } from "./types";

export function TabBar({
  openTabs,
  selectedPath,
  pendingChangesByPath,
  specDraftByPath,
  onOpenFile,
  onCloseTab,
}: {
  openTabs: string[];
  selectedPath: string | null;
  pendingChangesByPath: Record<string, PendingFileChange>;
  specDraftByPath: Record<string, string>;
  onOpenFile: (path: string) => void;
  onCloseTab: (path: string) => void;
}) {
  return (
    <div className="flex h-7 shrink-0 items-center border-b border-edge-default bg-surface-raised">
      <div className="flex min-w-0 flex-1 items-start self-stretch overflow-x-auto overflow-y-hidden">
        {openTabs.map((path) => {
          const change = pendingChangesByPath[path];
          const hasSpecDraft = specDraftByPath[path] !== undefined;
          const active = selectedPath === path;

          return (
            <div
              key={path}
              className={
                active
                  ? "flex h-7 min-w-0 max-w-56 items-center border-r border-edge-default bg-surface-subtle text-xs text-content-primary"
                  : "flex h-7 min-w-0 max-w-56 items-center border-r border-edge-subtle text-xs text-content-secondary hover:bg-surface-subtle hover:text-content-primary"
              }
            >
              <button
                type="button"
                className="flex h-full min-w-0 flex-1 items-center gap-1.5 px-2.5 text-left"
                onClick={() => onOpenFile(path)}
              >
                {change || hasSpecDraft ? (
                  <span className="size-1.5 shrink-0 rounded-full bg-orange-500" aria-hidden />
                ) : null}
                <span className="min-w-0 truncate">{path}</span>
              </button>
              <button
                type="button"
                aria-label={`Close ${path}`}
                className="mr-1 flex size-4 shrink-0 items-center justify-center rounded text-content-muted hover:bg-action-neutral-hover hover:text-content-primary"
                onClick={() => onCloseTab(path)}
              >
                <X className="h-3 w-3" />
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
