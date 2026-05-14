import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { parseAgentContent, type Segment } from "./parser";
import { ButtonsWidget } from "./ButtonsWidget";
import { ConfirmWidget } from "./ConfirmWidget";
import { ChartWidget } from "./ChartWidget";
import { CollapseWidget } from "./CollapseWidget";
import { StepsWidget } from "./StepsWidget";
import { BannerWidget } from "./BannerWidget";
import { MermaidWidget } from "./MermaidWidget";
import { CodeBlockWidget } from "./CodeBlockWidget";

const MARKDOWN_CLASSES =
  "max-w-none [&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-base [&_h1]:font-semibold [&_h1:first-child]:mt-0 " +
  "[&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-sm [&_h2]:font-semibold [&_h2:first-child]:mt-0 " +
  "[&_h3]:mb-0.5 [&_h3]:mt-1 [&_h3]:text-sm [&_h3]:font-semibold [&_h3:first-child]:mt-0 " +
  "[&_p]:mb-2 [&_p]:leading-relaxed [&_p:last-child]:mb-0 " +
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-0.5 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 " +
  "[&_code]:rounded [&_code]:bg-slate-200/70 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-200/70 [&_pre]:p-2 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current " +
  "[&_table]:my-4 [&_table]:w-full [&_table]:text-xs [&_table]:border-collapse " +
  "[&_table]:rounded-lg [&_table]:overflow-hidden [&_table]:border [&_table]:border-violet-200 [&_table]:bg-white [&_table]:shadow-sm " +
  "[&_thead]:bg-slate-50 [&_th]:px-3 [&_th]:py-1.5 [&_th]:text-left [&_th]:font-semibold [&_th]:text-slate-700 " +
  "[&_th]:border-b [&_th]:border-slate-200 " +
  "[&_td]:px-3 [&_td]:py-1.5 [&_td]:text-slate-600 [&_td]:border-b [&_td]:border-slate-100 " +
  "[&_tbody_tr:nth-child(even)]:bg-slate-50/60 " +
  "[&_tr:last-child_td]:border-b-0 [&_tr:hover]:bg-violet-50/50";

interface RichMessageProps {
  content: string;
  onAction?: (text: string) => void;
}

export function RichMessage({ content, onAction }: RichMessageProps) {
  const segments = parseAgentContent(content);

  return (
    <div>
      {segments.map((segment, i) => (
        <SegmentRenderer key={i} segment={segment} onAction={onAction} />
      ))}
    </div>
  );
}

function SegmentRenderer({ segment, onAction }: { segment: Segment; onAction?: (text: string) => void }) {
  switch (segment.type) {
    case "markdown":
      return (
        <div className={MARKDOWN_CLASSES}>
          <ReactMarkdown
            remarkPlugins={[remarkGfm, remarkBreaks]}
            components={{
              a: ({ children, href }) => (
                <a href={href} target="_blank" rel="noopener noreferrer">
                  {children}
                </a>
              ),
              code: ({ className, children, ...props }) => {
                const match = /language-(\w+)/.exec(className || "");
                const codeStr = String(children).replace(/\n$/, "");
                // Only use CodeBlockWidget for fenced code blocks (with language class)
                if (match) {
                  return <CodeBlockWidget code={codeStr} language={match[1]} />;
                }
                // Inline code
                return <code className={className} {...props}>{children}</code>;
              },
              pre: ({ children }) => <>{children}</>,
            }}
          >
            {segment.content}
          </ReactMarkdown>
        </div>
      );
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
    case "success":
      return <BannerWidget variant="success" content={segment.content} />;
    case "error":
      return <BannerWidget variant="error" content={segment.content} />;
  }
}
