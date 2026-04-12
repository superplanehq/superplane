import type { AiBuilderMessage } from "@/ui/BuildingBlocksSidebar/agentChat";
import { Activity, User } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";

export type AiMessageProps = {
  message: AiBuilderMessage;
};

export function AiMessage({ message }: AiMessageProps) {
  const isEmptyAssistantPlaceholder = message.role === "assistant" && message.content.trim().length === 0;
  if (isEmptyAssistantPlaceholder) {
    return null;
  }

  const isToolMessage = message.role === "tool";
  const isRunningToolMessage = isToolMessage && message.toolStatus === "running";

  let messageClassName = "";
  let wrapperClassName = "w-full";

  if (message.role === "user") {
    messageClassName =
      "flex w-full items-start gap-2 rounded-md border border-slate-200/90 bg-slate-100 px-3 py-2.5 text-sm text-slate-800";
    wrapperClassName = "w-full py-1";
  } else if (isToolMessage) {
    messageClassName = `flex items-start gap-2 px-2 text-xs leading-relaxed text-gray-500 ${isRunningToolMessage ? "sp-ai-thinking" : ""}`;
  } else {
    messageClassName = "px-2 text-sm text-gray-800";
  }

  return (
    <div key={message.id} className={wrapperClassName}>
      {message.role === "user" ? (
        <div className={messageClassName}>
          <User className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-500" aria-hidden="true" />
          <span className="min-w-0 whitespace-pre-wrap break-words">{message.content}</span>
        </div>
      ) : isToolMessage ? (
        <div className={messageClassName}>
          <Activity className="mt-0.5 h-3.5 w-3.5 shrink-0 text-gray-400" aria-hidden="true" />
          <span className="min-w-0 whitespace-pre-wrap break-words">{message.content}</span>
        </div>
      ) : (
        <div className={messageClassName}>
          {message.role === "assistant" ? <AiMessageMarkdown content={message.content} /> : message.content}
        </div>
      )}
    </div>
  );
}

function AiMessageMarkdown({ content }: { content: string }) {
  return (
    <div className="max-w-none text-slate-800 [&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-lg [&_h1]:font-semibold [&_h1]:leading-tight [&_h1:first-child]:mt-0 [&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-base [&_h2]:font-semibold [&_h2]:leading-tight [&_h2:first-child]:mt-0 [&_h3]:mb-1.5 [&_h3]:mt-2 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:leading-tight [&_h3:first-child]:mt-0 [&_h4]:mb-0.5 [&_h4]:mt-1 [&_h4]:text-sm [&_h4]:font-medium [&_h4]:leading-tight [&_h4:first-child]:mt-0 [&_p]:mb-2 [&_p]:leading-relaxed [&_p:last-child]:mb-0 [&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1 [&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 [&_hr]:my-6 [&_hr]:border-slate-300 [&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 [&_pre_code]:bg-transparent [&_pre_code]:p-0 [&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        components={{
          a: ({ children, href }) => (
            <a href={href} target="_blank" rel="noopener noreferrer" className="underline">
              {children}
            </a>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
