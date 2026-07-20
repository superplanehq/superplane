import { useCallback, useState, type ComponentProps, type ReactNode } from "react";
import ReactMarkdown, { defaultUrlTransform } from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { ClipboardList, ChevronDown, ChevronUp, X } from "lucide-react";
import type { RubricCategory } from "./parser";
import { IntegrationButton } from "./IntegrationButton";
import { MarkdownCode } from "./MarkdownCode";
import { NodeChipFromLink } from "./NodeChip";
import { RunChipFromLink } from "./RunChip";

const CRITERION_MARKDOWN_CLASSES =
  "[&_p]:m-0 [&_p]:inline " +
  "[&_ul]:my-1 [&_ul]:ml-4 [&_ul]:list-disc [&_ol]:my-1 [&_ol]:ml-4 [&_ol]:list-decimal [&_li]:my-0 " +
  "[&_strong]:font-semibold [&_em]:italic " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:text-slate-700 dark:[&_a]:text-gray-200 " +
  "[&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[0.85em] [&_code]:font-mono dark:[&_code]:bg-gray-700 " +
  "[&_pre]:my-1 [&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 [&_pre]:text-[11px] dark:[&_pre]:bg-gray-900/80 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_table]:w-full [&_table]:text-[11px] [&_table]:border-collapse " +
  "[&_thead]:bg-slate-50 dark:[&_thead]:bg-gray-900/60 [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:font-semibold [&_th]:text-slate-700 dark:[&_th]:text-gray-200 " +
  "[&_th]:border-b [&_th]:border-slate-200 dark:[&_th]:border-gray-700 " +
  "[&_td]:px-2 [&_td]:py-1 [&_td]:text-slate-600 dark:[&_td]:text-gray-300 [&_td]:border-b [&_td]:border-slate-100 dark:[&_td]:border-gray-700 " +
  "[&_tbody_tr:nth-child(even)]:bg-slate-50/60 dark:[&_tbody_tr:nth-child(even)]:bg-gray-900/40 " +
  "[&_tr:last-child_td]:border-b-0";

const RUBRIC_BODY_MARKDOWN_CLASSES =
  "max-w-none " +
  "[&_h2]:mb-2 [&_h2]:mt-3 [&_h2]:text-sm [&_h2]:font-semibold [&_h2:first-child]:mt-0 " +
  "[&_h3]:mb-1.5 [&_h3]:mt-2 [&_h3]:text-sm [&_h3]:font-semibold [&_h3:first-child]:mt-0 " +
  "[&_p]:mb-2 [&_p]:leading-relaxed [&_p:last-child]:mb-0 " +
  "[&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_li]:mb-1 " +
  "[&_strong]:font-semibold [&_em]:italic " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 dark:[&_blockquote]:border-gray-600 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:text-slate-700 dark:[&_a]:text-gray-200 " +
  "[&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[0.85em] [&_code]:font-mono dark:[&_code]:bg-gray-700 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_table]:w-full [&_table]:text-[11px] [&_table]:border-collapse " +
  "[&_thead]:bg-slate-50 dark:[&_thead]:bg-gray-900/60 [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:font-semibold [&_th]:text-slate-700 dark:[&_th]:text-gray-200 " +
  "[&_th]:border-b [&_th]:border-slate-200 dark:[&_th]:border-gray-700 " +
  "[&_td]:px-2 [&_td]:py-1 [&_td]:text-slate-600 dark:[&_td]:text-gray-300 [&_td]:border-b [&_td]:border-slate-100 dark:[&_td]:border-gray-700 " +
  "[&_tbody_tr:nth-child(even)]:bg-slate-50/60 dark:[&_tbody_tr:nth-child(even)]:bg-gray-900/40 " +
  "[&_tr:last-child_td]:border-b-0";

function RubricMarkdown({
  children,
  canvasId,
  organizationId,
  compact = false,
}: {
  children: string;
  canvasId?: string;
  organizationId?: string;
  compact?: boolean;
}) {
  return (
    <div className={`min-w-0 ${compact ? CRITERION_MARKDOWN_CLASSES : RUBRIC_BODY_MARKDOWN_CLASSES}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        urlTransform={(url) => (isAgentLink(url) ? url : defaultUrlTransform(url))}
        components={{
          a: ({ children: linkChildren, href }) => (
            <AgentLink href={href} canvasId={canvasId} organizationId={organizationId}>
              {linkChildren}
            </AgentLink>
          ),
          code: MarkdownCode,
          pre: ({ children: preChildren }) => <>{preChildren}</>,
          table: ({ children: tableChildren, ...props }) => (
            <div className="my-4 overflow-x-auto rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800">
              <table {...props}>{tableChildren}</table>
            </div>
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}

export interface RubricCriterion {
  text: string;
}

interface RubricWidgetProps {
  title: string;
  criteria: RubricCriterion[];
  categories?: RubricCategory[];
  onAction?: (text: string) => void;
  onStartBuilding?: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => void;
  canvasId?: string;
  organizationId?: string;
}

export function RubricWidget({
  title,
  criteria,
  categories,
  onAction,
  onStartBuilding,
  canvasId,
  organizationId,
}: RubricWidgetProps) {
  const [modalOpen, setModalOpen] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const totalCriteria = criteria.length;
  const hasCategories = categories && categories.length > 0;
  const rubricTitle = title || "Build Plan";

  // For preview: show first category or first 3 flat criteria
  const PREVIEW_COUNT = 3;
  const previewCriteria = hasCategories
    ? categories[0].criteria.slice(0, PREVIEW_COUNT)
    : criteria.slice(0, PREVIEW_COUNT);
  const hiddenCount = totalCriteria - previewCriteria.length;

  const handleStartBuilding = useCallback(() => {
    if (onStartBuilding) {
      onStartBuilding({ title, criteria: criteria.map((c) => c.text), categories });
      return;
    }

    onAction?.("Start building based on this plan");
  }, [categories, criteria, onAction, onStartBuilding, title]);

  const openModal = useCallback(() => {
    setModalOpen(true);
  }, []);

  const closeModal = useCallback(() => {
    setModalOpen(false);
  }, []);

  const expandPreview = useCallback(() => {
    setExpanded(true);
  }, []);

  const collapsePreview = useCallback(() => {
    setExpanded(false);
  }, []);

  return (
    <>
      <div className="my-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800">
        {/* Header */}
        <div className="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900/60">
          <ClipboardList size={14} className="shrink-0 text-slate-600 dark:text-gray-300" />
          <p className="flex-1 text-xs font-semibold text-slate-900 dark:text-gray-100">{rubricTitle}</p>
          <span className="text-[10px] font-medium text-slate-500 dark:text-gray-400">
            {hasCategories ? `${categories.length} sections · ` : ""}
            {totalCriteria} criteria
          </span>
        </div>

        {/* Preview */}
        <RubricPreview
          categories={categories}
          criteria={criteria}
          expanded={expanded}
          hasCategories={hasCategories}
          hiddenCount={hiddenCount}
          previewCriteria={previewCriteria}
          onExpand={expandPreview}
          onCollapse={collapsePreview}
          canvasId={canvasId}
          organizationId={organizationId}
        />

        {/* Actions */}
        <div className="flex items-center gap-2 border-t border-slate-100 px-3 pb-3 pt-1 dark:border-gray-700">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 text-xs text-slate-500 dark:text-gray-400"
            onClick={openModal}
          >
            View Full Plan
          </Button>
          <Button size="sm" className="ml-auto h-7 text-xs" onClick={handleStartBuilding}>
            Start Building →
          </Button>
        </div>
      </div>

      {/* Modal */}
      {modalOpen && (
        <RubricModal
          title={rubricTitle}
          criteria={criteria}
          categories={categories}
          hasCategories={hasCategories}
          onClose={closeModal}
          canvasId={canvasId}
          organizationId={organizationId}
        />
      )}
    </>
  );
}

function RubricPreview({
  categories,
  criteria,
  expanded,
  hasCategories,
  hiddenCount,
  previewCriteria,
  onExpand,
  onCollapse,
  canvasId,
  organizationId,
}: {
  categories?: RubricCategory[];
  criteria: RubricCriterion[];
  expanded: boolean;
  hasCategories: boolean | undefined;
  hiddenCount: number;
  previewCriteria: RubricCriterion[];
  onExpand: () => void;
  onCollapse: () => void;
  canvasId?: string;
  organizationId?: string;
}) {
  return (
    <div className="px-3 py-2">
      {hasCategories && !expanded ? (
        <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
          {categories?.[0]?.heading}
        </p>
      ) : null}

      {!expanded ? (
        <FlatCriteriaList criteria={previewCriteria} canvasId={canvasId} organizationId={organizationId} />
      ) : null}
      {expanded && hasCategories ? (
        <CategorizedList
          categories={categories ?? []}
          showNumbers
          canvasId={canvasId}
          organizationId={organizationId}
        />
      ) : null}
      {expanded && !hasCategories ? (
        <FlatCriteriaList criteria={criteria} canvasId={canvasId} organizationId={organizationId} />
      ) : null}

      {hiddenCount > 0 && !expanded ? (
        <PreviewToggleButton direction="down" onClick={onExpand}>
          +{hiddenCount} more
        </PreviewToggleButton>
      ) : null}
      {expanded && hiddenCount > 0 ? (
        <PreviewToggleButton direction="up" onClick={onCollapse}>
          Show less
        </PreviewToggleButton>
      ) : null}
    </div>
  );
}

function RubricModal({
  title,
  criteria,
  categories,
  hasCategories,
  onClose,
  canvasId,
  organizationId,
}: {
  title: string;
  criteria: RubricCriterion[];
  categories?: RubricCategory[];
  hasCategories: boolean | undefined;
  onClose: () => void;
  canvasId?: string;
  organizationId?: string;
}) {
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent
        showCloseButton={false}
        className="flex max-h-[80vh] w-full max-w-lg flex-col gap-0 overflow-hidden p-0"
      >
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-gray-700">
          <div className="flex items-center gap-2">
            <ClipboardList size={16} className="text-slate-600 dark:text-gray-300" />
            <DialogTitle className="text-sm font-semibold text-slate-900 dark:text-gray-100">{title}</DialogTitle>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 dark:text-gray-500 dark:hover:text-gray-300"
          >
            <X size={16} />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          {hasCategories ? (
            <CategorizedList
              categories={categories ?? []}
              showNumbers
              canvasId={canvasId}
              organizationId={organizationId}
            />
          ) : (
            <NumberedCriteriaList criteria={criteria} canvasId={canvasId} organizationId={organizationId} />
          )}
        </div>
        <div className="flex justify-end border-t border-slate-200 px-4 py-3 dark:border-gray-700">
          <Button variant="ghost" size="sm" className="text-xs" onClick={onClose}>
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function PreviewToggleButton({
  direction,
  children,
  onClick,
}: {
  direction: "down" | "up";
  children: ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="mt-1 flex items-center gap-1 text-[10px] text-slate-500 hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-200"
    >
      {direction === "down" ? <ChevronDown size={10} /> : <ChevronUp size={10} />}
      {children}
    </button>
  );
}

function FlatCriteriaList({
  criteria,
  canvasId,
  organizationId,
}: {
  criteria: RubricCriterion[];
  canvasId?: string;
  organizationId?: string;
}) {
  return criteria.map((criterion, index) => (
    <div key={index} className="flex items-start gap-2 py-0.5">
      <span className="mt-0.5 shrink-0 text-xs text-slate-400 dark:text-gray-500">✦</span>
      <div className="min-w-0 flex-1 text-xs text-slate-700 dark:text-gray-300">
        <RubricMarkdown compact canvasId={canvasId} organizationId={organizationId}>
          {criterion.text}
        </RubricMarkdown>
      </div>
    </div>
  ));
}

function NumberedCriteriaList({
  criteria,
  canvasId,
  organizationId,
}: {
  criteria: RubricCriterion[];
  canvasId?: string;
  organizationId?: string;
}) {
  return criteria.map((criterion, index) => (
    <div
      key={index}
      className="flex items-start gap-2 border-b border-slate-50 py-1.5 last:border-0 dark:border-gray-700"
    >
      <span className="mt-0.5 shrink-0 text-sm font-medium text-slate-500 dark:text-gray-400">{index + 1}.</span>
      <div className="min-w-0 flex-1 text-sm text-slate-700 dark:text-gray-300">
        <RubricMarkdown compact canvasId={canvasId} organizationId={organizationId}>
          {criterion.text}
        </RubricMarkdown>
      </div>
    </div>
  ));
}

function CategorizedList({
  categories,
  showNumbers,
  canvasId,
  organizationId,
}: {
  categories: RubricCategory[];
  showNumbers?: boolean;
  canvasId?: string;
  organizationId?: string;
}) {
  let globalIndex = 0;
  return (
    <div className="space-y-3">
      {categories.map((cat, ci) => {
        const categoryStartIndex = globalIndex;
        globalIndex += cat.criteria.length;

        return (
          <div key={ci}>
            <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
              {cat.heading}
            </p>
            {cat.body ? (
              <div className={`${showNumbers ? "text-sm" : "text-xs"} text-slate-700 dark:text-gray-300`}>
                <RubricMarkdown canvasId={canvasId} organizationId={organizationId}>
                  {cat.body}
                </RubricMarkdown>
              </div>
            ) : (
              cat.criteria.map((c, i) => {
                const criterionIndex = categoryStartIndex + i + 1;
                return (
                  <div
                    key={i}
                    className={`flex items-start gap-2 ${
                      showNumbers ? "border-b border-slate-50 py-1.5 last:border-0 dark:border-gray-700" : "py-0.5"
                    }`}
                  >
                    {showNumbers ? (
                      <span className="mt-0.5 shrink-0 text-sm font-medium text-slate-500 dark:text-gray-400">
                        {criterionIndex}.
                      </span>
                    ) : (
                      <span className="mt-0.5 shrink-0 text-xs text-slate-400 dark:text-gray-500">✦</span>
                    )}
                    <div
                      className={`min-w-0 flex-1 ${showNumbers ? "text-sm" : "text-xs"} text-slate-700 dark:text-gray-300`}
                    >
                      <RubricMarkdown compact canvasId={canvasId} organizationId={organizationId}>
                        {c.text}
                      </RubricMarkdown>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        );
      })}
    </div>
  );
}

function isAgentLink(url: string): boolean {
  return url.startsWith("run:") || url.startsWith("node:") || url.startsWith("integration:");
}

function AgentLink({
  href,
  children,
  canvasId,
  organizationId,
}: ComponentProps<"a"> & { canvasId?: string; organizationId?: string }) {
  const specialLink = renderSpecialLink(href, children, canvasId, organizationId);
  if (specialLink) {
    return specialLink;
  }

  return (
    <a href={href} target="_blank" rel="noopener noreferrer">
      {children}
    </a>
  );
}

function renderSpecialLink(href: string | undefined, children: ReactNode, canvasId?: string, organizationId?: string) {
  const label = typeof children === "string" ? children : undefined;

  const runMatch = href?.match(/^run:([0-9a-f-]{36})(?:~(.+))?/);
  if (runMatch && canvasId && organizationId) {
    return (
      <RunChipFromLink
        runId={runMatch[1]}
        rawLabel={label}
        rawStatus={runMatch[2]}
        canvasId={canvasId}
        organizationId={organizationId}
      />
    );
  }

  const integrationMatch = href?.match(/^integration:(.+)$/);
  if (integrationMatch) {
    return <IntegrationButton integrationRef={integrationMatch[1]} label={label} />;
  }

  const nodeMatch = href?.match(/^node:(.+)$/);
  if (nodeMatch && canvasId && organizationId) {
    return (
      <NodeChipFromLink nodeId={nodeMatch[1]} rawLabel={label} canvasId={canvasId} organizationId={organizationId} />
    );
  }

  return null;
}
