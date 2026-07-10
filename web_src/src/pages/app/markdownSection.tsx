import { Children, createContext, isValidElement, useContext, useId, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";

import { splitBlockquoteMarkerLine } from "./markdownBlockquoteMarker";
import { resolveMarkdownSectionPreset } from "./markdownSectionPresets";

const SECTION_MARKER_RE = /^\[!SECTION(?::([a-z0-9_-]+))?\]\s+(.+)$/i;

type ParsedSection = {
  title: string;
  trailing?: string;
  presetId?: string;
  body: ReactNode;
};

const SectionDepthContext = createContext(0);

/**
 * If `children` are a `[!SECTION] Title` or `[!SECTION:preset] Title` blockquote,
 * return the title/preset/body with the marker line removed.
 *
 * Optional trailing meta after a middle-dot separator:
 * `> [!SECTION:rules] Rules · ~5,366`
 */
export function parseGithubSectionChildren(children: ReactNode): ParsedSection | null {
  const split = splitBlockquoteMarkerLine(children);
  if (!split) {
    return null;
  }

  const match = split.markerText.match(SECTION_MARKER_RE);
  if (!match) {
    return null;
  }

  const presetId = match[1]?.toLowerCase();
  const rawTitle = match[2].trim();
  if (!rawTitle) {
    return null;
  }

  const { title, trailing } = splitSectionTitle(rawTitle);
  return { title, trailing, presetId, body: split.body };
}

export function MarkdownSection({
  title,
  trailing,
  presetId,
  children,
}: {
  title: string;
  trailing?: string;
  presetId?: string;
  children: ReactNode;
}) {
  const depth = useContext(SectionDepthContext);
  const [open, setOpen] = useState(false);
  const panelId = useId();
  const isRoot = depth === 0;
  const preset = resolveMarkdownSectionPreset(presetId, depth);
  const Icon = preset.Icon;
  const childSectionCount = useMemo(() => countDirectChildSections(children), [children]);

  return (
    <div
      className={cn("min-w-0", isRoot ? "my-2" : "my-0.5")}
      data-testid="markdown-section"
      data-section-preset={preset.id}
      data-section-count={childSectionCount > 0 ? String(childSectionCount) : undefined}
    >
      <div className={cn("min-w-0 rounded-lg", isRoot && preset.barClassName)}>
        <button
          type="button"
          aria-expanded={open}
          aria-controls={panelId}
          className={cn(
            "flex w-full min-w-0 items-center gap-2 px-2.5 text-left",
            isRoot ? "min-h-9" : "min-h-8 rounded-md hover:bg-slate-100/70 dark:hover:bg-gray-800/60",
          )}
          onClick={() => setOpen((current) => !current)}
        >
          <span className="flex size-5 shrink-0 items-center justify-center">
            <ChevronRight
              className={cn(
                "size-3.5 text-slate-500 transition-transform duration-200 dark:text-gray-400",
                open && "rotate-90",
              )}
              aria-hidden
            />
          </span>
          <span className="flex size-5 shrink-0 items-center justify-center">
            <Icon className={cn("size-3.5", preset.iconClassName)} aria-hidden />
          </span>
          <span className="flex min-w-0 flex-1 items-baseline gap-1.5">
            <span className="truncate text-[13px] font-semibold text-slate-900 dark:text-gray-100">{title}</span>
            {childSectionCount > 0 ? (
              <span
                className="shrink-0 text-[12px] font-normal tabular-nums text-slate-500 dark:text-gray-400"
                data-testid="markdown-section-count"
              >
                {childSectionCount}
              </span>
            ) : null}
          </span>
          {trailing ? (
            <span className="shrink-0 text-[12px] tabular-nums text-slate-500 dark:text-gray-400">{trailing}</span>
          ) : null}
        </button>
      </div>

      {open ? (
        <div id={panelId} role="region" className={cn("min-w-0 pt-1", isRoot ? "pl-2" : "pl-0")}>
          <SectionDepthContext.Provider value={depth + 1}>
            <div
              className={cn(
                "min-w-0 border-l border-slate-200/90 pl-3 dark:border-gray-700/80",
                "text-[13px] leading-relaxed text-slate-600 dark:text-gray-300",
                "[&_p]:mb-2 [&_p:last-child]:mb-0",
                "[&_[data-testid=markdown-section]]:ml-1",
              )}
            >
              {children}
            </div>
          </SectionDepthContext.Provider>
        </div>
      ) : null}
    </div>
  );
}

/**
 * Count nested `[!SECTION]` blocks that are direct children of this section’s
 * body (walking through wrapper elements, but not into other sections).
 *
 * Nested quotes arrive from react-markdown as unevaluated `blockquote`
 * components (not yet `MarkdownSection`), so we detect them by parsing their
 * children for a `[!SECTION]` marker.
 */
export function countDirectChildSections(children: ReactNode): number {
  let count = 0;

  Children.forEach(children, (child) => {
    if (!isValidElement<{ children?: ReactNode }>(child)) {
      return;
    }

    if (child.type === MarkdownSection) {
      count += 1;
      return;
    }

    // Unevaluated nested blockquote that will render as a section.
    if (child.props.children != null && parseGithubSectionChildren(child.props.children)) {
      count += 1;
      return;
    }

    if (child.props.children != null) {
      count += countDirectChildSections(child.props.children);
    }
  });

  return count;
}

function splitSectionTitle(rawTitle: string): { title: string; trailing?: string } {
  const separator = " · ";
  const separatorIndex = rawTitle.indexOf(separator);
  if (separatorIndex < 0) {
    return { title: rawTitle };
  }

  const title = rawTitle.slice(0, separatorIndex).trim();
  const trailing = rawTitle.slice(separatorIndex + separator.length).trim();
  if (!title) {
    return { title: rawTitle };
  }

  return trailing ? { title, trailing } : { title };
}
