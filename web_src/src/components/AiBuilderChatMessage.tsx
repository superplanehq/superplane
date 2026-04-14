import { AiBuilderIntegrationActions } from "@/components/AiBuilderIntegrationActions";
import { AiBuilderOptionChips } from "@/components/AiBuilderOptionChips";
import { MermaidDiagram } from "@/components/MermaidDiagram";
import type { AiBuilderMessage } from "@/ui/BuildingBlocksSidebar/agentChat";
import "highlight.js/styles/github.css";
import {
  Activity,
  AlertCircle,
  AlertTriangle,
  Check,
  ChevronRight,
  Clipboard,
  ExternalLink,
  Flame,
  Info,
  Lightbulb,
} from "lucide-react";
import {
  Children,
  type ComponentPropsWithoutRef,
  type ReactNode,
  isValidElement,
  useCallback,
  useRef,
  useState,
} from "react";
import ReactMarkdown from "react-markdown";
import rehypeHighlight from "rehype-highlight";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { cn } from "../lib/utils";

export type AiMessageProps = {
  message: AiBuilderMessage;
  isLastAssistant?: boolean;
  isGeneratingResponse?: boolean;
  onSendPrompt?: (value: string) => void;
  onConnectIntegration?: (integrationName: string) => void;
  onFocusInput?: () => void;
  connectedIntegrationNames?: Set<string>;
};

export function AiMessage({
  message,
  isLastAssistant = false,
  isGeneratingResponse = false,
  onSendPrompt,
  onConnectIntegration,
  onFocusInput,
  connectedIntegrationNames,
}: AiMessageProps) {
  if (message.role === "assistant" && message.content.trim().length === 0) {
    return null;
  }

  switch (message.role) {
    case "user":
      return <UserMessage content={message.content} />;
    case "tool":
      return <ToolMessage message={message} />;
    case "assistant":
      return (
        <AssistantMessage
          message={message}
          isLastAssistant={isLastAssistant}
          isGeneratingResponse={isGeneratingResponse}
          onSendPrompt={onSendPrompt}
          onConnectIntegration={onConnectIntegration}
          onFocusInput={onFocusInput}
          connectedIntegrationNames={connectedIntegrationNames}
        />
      );
    default:
      return null;
  }
}

function ToolMessage({ message }: { message: AiBuilderMessage }) {
  const isRunning = message.toolStatus === "running";

  const className = cn(
    "flex items-center gap-2 px-2 text-xs leading-relaxed text-gray-500",
    isRunning ? "sp-ai-thinking" : "",
  );

  return (
    <div className="w-full">
      <div className={className}>
        <Activity className="h-3 w-3 shrink-0 text-gray-400" aria-hidden="true" />
        <span className="min-w-0 whitespace-pre-wrap break-words">{message.content}</span>
      </div>
    </div>
  );
}

function UserMessage({ content }: { content: string }) {
  return (
    <div className="w-full py-1">
      <div className="flex w-full items-start gap-2 rounded-sm border border-slate-200/90 bg-slate-100 px-2 py-1.5 text-sm text-slate-800">
        <span className="min-w-0 whitespace-pre-wrap break-words">{content}</span>
      </div>
    </div>
  );
}

function AssistantMessage({
  message,
  isLastAssistant,
  isGeneratingResponse,
  onSendPrompt,
  onConnectIntegration,
  onFocusInput,
  connectedIntegrationNames,
}: {
  message: AiBuilderMessage;
  isLastAssistant: boolean;
  isGeneratingResponse: boolean;
  onSendPrompt?: (value: string) => void;
  onConnectIntegration?: (integrationName: string) => void;
  onFocusInput?: () => void;
  connectedIntegrationNames?: Set<string>;
}) {
  const hasFollowUpOptions = isLastAssistant && (message.followUpOptions?.length ?? 0) > 0;
  const hasIntegrationActions = isLastAssistant && (message.integrationActions?.length ?? 0) > 0;

  return (
    <div className="w-full">
      <div className="px-2 text-sm text-gray-800">
        <AiMessageMarkdown content={message.content} />
        {hasIntegrationActions && onConnectIntegration ? (
          <AiBuilderIntegrationActions
            actions={message.integrationActions!}
            onConnect={onConnectIntegration}
            disabled={isGeneratingResponse}
            connectedIntegrationNames={connectedIntegrationNames}
          />
        ) : null}
        {hasFollowUpOptions && onSendPrompt ? (
          <AiBuilderOptionChips
            options={message.followUpOptions!}
            onSelect={onSendPrompt}
            onFocusInput={onFocusInput ?? (() => {})}
            disabled={isGeneratingResponse}
          />
        ) : null}
      </div>
    </div>
  );
}

const sanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "details", "summary"],
  attributes: {
    ...defaultSchema.attributes,
    details: [...(defaultSchema.attributes?.details ?? []), "open"],
    code: [...(defaultSchema.attributes?.code ?? []), "className"],
  },
};

const MARKDOWN_CLASSES = [
  "max-w-none text-slate-800",
  "[&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-lg [&_h1]:font-semibold [&_h1]:leading-tight [&_h1:first-child]:mt-0",
  "[&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-base [&_h2]:font-semibold [&_h2]:leading-tight [&_h2:first-child]:mt-0",
  "[&_h3]:mb-1.5 [&_h3]:mt-2 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:leading-tight [&_h3:first-child]:mt-0",
  "[&_h4]:mb-0.5 [&_h4]:mt-1 [&_h4]:text-sm [&_h4]:font-medium [&_h4]:leading-tight [&_h4:first-child]:mt-0",
  "[&_p]:mb-2 [&_p]:leading-relaxed [&_p:last-child]:mb-0",
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1",
  "[&_hr]:my-6 [&_hr]:border-slate-300",
  "[&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs",
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2",
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0",
].join(" ");

const ADMONITION_TYPES = {
  NOTE: {
    icon: Info,
    title: "Note",
    border: "border-blue-400",
    bg: "bg-blue-50",
    text: "text-blue-800",
    iconColor: "text-blue-500",
  },
  TIP: {
    icon: Lightbulb,
    title: "Tip",
    border: "border-green-400",
    bg: "bg-green-50",
    text: "text-green-800",
    iconColor: "text-green-500",
  },
  IMPORTANT: {
    icon: AlertCircle,
    title: "Important",
    border: "border-purple-400",
    bg: "bg-purple-50",
    text: "text-purple-800",
    iconColor: "text-purple-500",
  },
  WARNING: {
    icon: AlertTriangle,
    title: "Warning",
    border: "border-amber-400",
    bg: "bg-amber-50",
    text: "text-amber-800",
    iconColor: "text-amber-500",
  },
  CAUTION: {
    icon: Flame,
    title: "Caution",
    border: "border-red-400",
    bg: "bg-red-50",
    text: "text-red-800",
    iconColor: "text-red-500",
  },
} as const;

type AdmonitionType = keyof typeof ADMONITION_TYPES;

function extractTextContent(node: ReactNode): string {
  if (typeof node === "string") return node;
  if (typeof node === "number") return String(node);
  if (!isValidElement(node)) return "";
  const children = (node.props as { children?: ReactNode }).children;
  if (!children) return "";
  return Children.toArray(children).map(extractTextContent).join("");
}

function stripAdmonitionTag(children: ReactNode, tag: string): ReactNode[] {
  const pattern = new RegExp(`\\[!${tag}\\]\\s*\\n?`);
  return Children.toArray(children).map((child) => {
    if (typeof child === "string") return child.replace(pattern, "");
    if (isValidElement(child)) {
      const props = child.props as { children?: ReactNode };
      if (props.children) {
        const stripped = stripAdmonitionTag(props.children, tag);
        return { ...child, props: { ...props, children: stripped } };
      }
    }
    return child;
  });
}

function MdBlockquote({ children, ...rest }: ComponentPropsWithoutRef<"blockquote">) {
  const text = Children.toArray(children).map(extractTextContent).join("");
  const match = text.match(/\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]/);
  if (!match) {
    return (
      <blockquote className="my-2 border-l-2 border-slate-300 pl-3 text-slate-600" {...rest}>
        {children}
      </blockquote>
    );
  }
  const type = match[1] as AdmonitionType;
  const config = ADMONITION_TYPES[type];
  const Icon = config.icon;
  const body = stripAdmonitionTag(children, type);

  return (
    <div className={cn("my-2 rounded-md border-l-4 px-3 py-2", config.border, config.bg)}>
      <div className={cn("mb-1 flex items-center gap-1.5 text-xs font-semibold", config.text)}>
        <Icon className={cn("h-3.5 w-3.5", config.iconColor)} />
        {config.title}
      </div>
      <div className={cn("text-xs leading-relaxed", config.text, "[&_p]:mb-1 [&_p:last-child]:mb-0")}>{body}</div>
    </div>
  );
}

function isMermaidCodeBlock(children: ReactNode): string | null {
  const child = Children.toArray(children)[0];
  if (!isValidElement(child)) return null;
  const props = child.props as { className?: string; children?: ReactNode };
  if (typeof props.className !== "string" || !props.className.includes("language-mermaid")) return null;
  const text = extractTextContent(child);
  return text.trim() || null;
}

function MdCodeBlock({ children, ...rest }: ComponentPropsWithoutRef<"pre">) {
  const codeRef = useRef<HTMLPreElement>(null);
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    const code = codeRef.current?.querySelector("code");
    if (!code) return;
    void navigator.clipboard.writeText(code.textContent ?? "").then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  }, []);

  const mermaidDef = isMermaidCodeBlock(children);
  if (mermaidDef) {
    return (
      <div className="my-2 overflow-hidden rounded-md border border-slate-200 bg-white p-3">
        <MermaidDiagram definition={mermaidDef} className="[&_svg]:max-w-full [&_svg]:h-auto [&_svg]:mx-auto" />
      </div>
    );
  }

  return (
    <div className="group relative my-2">
      <pre ref={codeRef} className="overflow-auto rounded-md bg-slate-100 p-3 text-xs leading-relaxed" {...rest}>
        {children}
      </pre>
      <button
        type="button"
        onClick={handleCopy}
        className="absolute right-1.5 top-1.5 rounded-md bg-white/80 p-1 text-slate-500 opacity-0 shadow-sm transition-opacity hover:bg-white hover:text-slate-700 group-hover:opacity-100"
        aria-label="Copy code"
      >
        {copied ? <Check className="h-3.5 w-3.5" /> : <Clipboard className="h-3.5 w-3.5" />}
      </button>
    </div>
  );
}

function MdTable({ children, ...rest }: ComponentPropsWithoutRef<"table">) {
  return (
    <div className="my-2 overflow-x-auto rounded-md border border-slate-200">
      <table
        className="min-w-full text-xs [&_td]:border-t [&_td]:border-slate-200 [&_td]:px-2.5 [&_td]:py-1.5 [&_th]:bg-slate-50 [&_th]:px-2.5 [&_th]:py-1.5 [&_th]:text-left [&_th]:font-semibold [&_tr:nth-child(even)]:bg-slate-50/50"
        {...rest}
      >
        {children}
      </table>
    </div>
  );
}

function MdLink({ children, href, ...rest }: ComponentPropsWithoutRef<"a">) {
  const isExternal = href && (href.startsWith("http://") || href.startsWith("https://"));
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-baseline gap-0.5 underline underline-offset-2 decoration-current"
      {...rest}
    >
      {children}
      {isExternal ? <ExternalLink className="inline h-2.5 w-2.5 shrink-0 self-center" /> : null}
    </a>
  );
}

function MdDetails({ children, ...rest }: ComponentPropsWithoutRef<"details">) {
  return (
    <details className="group/details my-2 rounded-md border border-slate-200 bg-white" {...rest}>
      {children}
    </details>
  );
}

function MdSummary({ children, ...rest }: ComponentPropsWithoutRef<"summary">) {
  return (
    <summary
      className="flex cursor-pointer select-none items-center gap-1.5 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50"
      {...rest}
    >
      <ChevronRight className="h-3 w-3 shrink-0 transition-transform group-open/details:rotate-90" />
      {children}
    </summary>
  );
}

function MdListItem({ children, ...rest }: ComponentPropsWithoutRef<"li">) {
  const childArray = Children.toArray(children);
  const hasCheckbox = childArray.some(
    (child) => isValidElement(child) && (child.props as { type?: string }).type === "checkbox",
  );
  if (hasCheckbox) {
    return (
      <li className="mb-1 flex list-none items-start gap-1.5" {...rest}>
        {children}
      </li>
    );
  }
  return (
    <li className="mb-1" {...rest}>
      {children}
    </li>
  );
}

function MdCheckbox(props: ComponentPropsWithoutRef<"input">) {
  const { checked } = props;
  return (
    <span
      className={cn(
        "mt-0.5 inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-sm border",
        checked ? "border-blue-500 bg-blue-500 text-white" : "border-slate-300 bg-white",
      )}
    >
      {checked ? <Check className="h-2.5 w-2.5" /> : null}
    </span>
  );
}

const markdownComponents = {
  a: MdLink,
  blockquote: MdBlockquote,
  pre: MdCodeBlock,
  table: MdTable,
  details: MdDetails,
  summary: MdSummary,
  li: MdListItem,
  input: MdCheckbox,
} as const;

function AiMessageMarkdown({ content }: { content: string }) {
  return (
    <div className={MARKDOWN_CLASSES}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        rehypePlugins={[rehypeRaw, [rehypeSanitize, sanitizeSchema], rehypeHighlight]}
        components={markdownComponents}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
