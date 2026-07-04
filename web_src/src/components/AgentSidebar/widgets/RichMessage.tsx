import { memo, useMemo, type ComponentProps, type ReactNode } from "react";
import ReactMarkdown, { defaultUrlTransform } from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { BannerWidget } from "./BannerWidget";
import { ButtonsWidget } from "./ButtonsWidget";
import { ChartWidget } from "./ChartWidget";
import { CollapseWidget } from "./CollapseWidget";
import { ConfirmWidget } from "./ConfirmWidget";
import { MarkdownCode } from "./MarkdownCode";
import { RubricWidget } from "./RubricWidget";
import { MermaidWidget } from "./MermaidWidget";
import { NodeChipFromLink } from "./NodeChip";
import { parseAgentContent, type RubricCategory, type Segment } from "./parser";
import { RunChipFromLink } from "./RunChip";
import { StepsWidget } from "./StepsWidget";
import { SurveyWidget } from "./SurveyWidget";
import { IntegrationButton } from "./IntegrationButton";

const MARKDOWN_CLASSES =
  "max-w-none [&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-base [&_h1]:font-semibold [&_h1:first-child]:mt-0 " +
  "[&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-sm [&_h2]:font-semibold [&_h2:first-child]:mt-0 " +
  "[&_h3]:mb-0.5 [&_h3]:mt-1 [&_h3]:text-sm [&_h3]:font-semibold [&_h3:first-child]:mt-0 " +
  "[&_p]:mb-2 [&_p]:leading-relaxed [&_p:last-child]:mb-0 " +
  "[&_strong]:font-semibold [&_b]:font-semibold " +
  "[&_hr]:my-5 [&_hr]:border-0 [&_hr]:border-t [&_hr]:border-slate-200 " +
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-0.5 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 " +
  "[&_code]:rounded [&_code]:bg-slate-200/70 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-200/70 [&_pre]:p-2 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current " +
  "[&_table]:w-full [&_table]:text-xs [&_table]:border-collapse " +
  "[&_thead]:bg-slate-50 [&_th]:px-3 [&_th]:py-1.5 [&_th]:text-left [&_th]:font-semibold [&_th]:text-slate-700 " +
  "[&_th]:border-b [&_th]:border-slate-200 " +
  "[&_td]:px-3 [&_td]:py-1.5 [&_td]:text-slate-600 [&_td]:border-b [&_td]:border-slate-100 " +
  "[&_tbody_tr:nth-child(even)]:bg-slate-50/60 " +
  "[&_tr:last-child_td]:border-b-0 [&_tr:hover]:bg-slate-50/50";

type StartBuildingRubric = {
  title: string;
  criteria: string[];
  categories?: RubricCategory[];
};

interface RichMessageProps {
  content: string;
  onAction?: (text: string) => void;
  onStartBuilding?: (rubric: StartBuildingRubric) => void;
  canvasId?: string;
  organizationId?: string;
}

export const RichMessage = memo(function RichMessage({
  content,
  onAction,
  onStartBuilding,
  canvasId,
  organizationId,
}: RichMessageProps) {
  // `parseAgentContent` + the downstream ReactMarkdown render are the most
  // expensive work in the sidebar. Memoize by content so parent re-renders
  // (canvas pan/zoom, WebSocket status ticks, etc.) don't redo it.
  const segments = useMemo(() => parseAgentContent(content), [content]);

  return (
    <div className="w-full min-w-0">
      {segments.map((segment, i) => (
        <SegmentRenderer
          key={i}
          segment={segment}
          onAction={onAction}
          onStartBuilding={onStartBuilding}
          canvasId={canvasId}
          organizationId={organizationId}
        />
      ))}
    </div>
  );
});

function SegmentRenderer({
  segment,
  onAction,
  onStartBuilding,
  canvasId,
  organizationId,
}: {
  segment: Segment;
  onAction?: (text: string) => void;
  onStartBuilding?: (rubric: StartBuildingRubric) => void;
  canvasId?: string;
  organizationId?: string;
}) {
  switch (segment.type) {
    case "markdown":
      return <MarkdownSegment content={segment.content} canvasId={canvasId} organizationId={organizationId} />;
    case "buttons":
      return <ButtonsWidget prompt={segment.prompt} items={segment.items} onAction={onAction} />;
    case "confirm":
      return <ConfirmWidget message={segment.message} yes={segment.yes} no={segment.no} onAction={onAction} />;
    case "chart":
      return <ChartWidget config={segment.config} />;
    case "collapse":
      return <CollapseWidget title={segment.title} content={segment.content} />;
    case "mermaid":
      return <MermaidWidget content={segment.content} />;
    case "steps":
      return <StepsWidget items={segment.items} />;
    case "survey":
      return <SurveyWidget questions={segment.questions} onAction={onAction} />;
    case "rubric":
      return (
        <RubricWidget
          title={segment.title}
          criteria={segment.criteria}
          categories={segment.categories}
          onAction={onAction}
          onStartBuilding={onStartBuilding}
          canvasId={canvasId}
          organizationId={organizationId}
        />
      );
    case "success":
      return <BannerWidget variant="success" content={segment.content} />;
    case "error":
      return <BannerWidget variant="error" content={segment.content} />;
    case "draft-actions":
      // Rendered externally as StagingActionsBar, not inline
      return null;
  }
}

function MarkdownSegment({
  content,
  canvasId,
  organizationId,
}: {
  content: string;
  canvasId?: string;
  organizationId?: string;
}) {
  return (
    <div className={`min-w-0 ${MARKDOWN_CLASSES}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        urlTransform={(url) => (isAgentLink(url) ? url : defaultUrlTransform(url))}
        components={{
          a: ({ children, href }) => (
            <AgentLink href={href} canvasId={canvasId} organizationId={organizationId}>
              {children}
            </AgentLink>
          ),
          code: MarkdownCode,
          pre: ({ children }) => <>{children}</>,
          table: ({ children, ...props }) => (
            <div className="my-4 overflow-x-auto rounded-lg border border-slate-200 bg-white">
              <table {...props}>{children}</table>
            </div>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
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

function isAgentLink(url: string): boolean {
  return url.startsWith("run:") || url.startsWith("node:") || url.startsWith("integration:");
}
