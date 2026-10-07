import { memo, useMemo, type ComponentProps, type ReactNode } from "react";
import ReactMarkdown, { defaultUrlTransform } from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { BannerWidget } from "@/components/AgentSidebar/widgets/BannerWidget";
import { ButtonsWidget } from "@/components/AgentSidebar/widgets/ButtonsWidget";
import { ChartWidget } from "@/components/AgentSidebar/widgets/ChartWidget";
import { CollapseWidget } from "@/components/AgentSidebar/widgets/CollapseWidget";
import { ConfirmWidget } from "@/components/AgentSidebar/widgets/ConfirmWidget";
import { MarkdownCode } from "@/components/AgentSidebar/widgets/MarkdownCode";
import { RubricWidget } from "@/components/AgentSidebar/widgets/RubricWidget";
import { MermaidWidget } from "@/components/AgentSidebar/widgets/MermaidWidget";
import { NodeChipFromLink } from "@/components/AgentSidebar/widgets/NodeChip";
import { parseAgentContent, type RubricCategory, type Segment } from "@/components/AgentSidebar/widgets/parser";
import { RunChipFromLink } from "@/components/AgentSidebar/widgets/RunChip";
import { StepsWidget } from "@/components/AgentSidebar/widgets/StepsWidget";
import { SurveyWidget } from "@/components/AgentSidebar/widgets/SurveyWidget";
import { IntegrationButton } from "@/components/AgentSidebar/widgets/IntegrationButton";
import { factoryAgentMessageClassName } from "./factoryAgentChrome";

const MARKDOWN_CLASSES = `workspace-markdown min-w-0 max-w-none text-foreground ${factoryAgentMessageClassName}`;

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

export const FactoryRichMessage = memo(function FactoryRichMessage({
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
            <div className="my-4 overflow-x-auto rounded-lg border border-border bg-card">
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
