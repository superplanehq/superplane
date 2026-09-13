import { MarkdownContent } from "@/pages/app/Markdown";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMessage } from "../createWithAgentTypes";

const MESSAGE_MARKDOWN =
  "max-w-none font-sans text-[14px] leading-5 text-foreground [&_p:first-child]:mt-0 [&_p:last-child]:mb-0";

export function WorkOrderIntentTranscript({ messages }: { messages: CreateWithAgentMessage[] }) {
  if (messages.length === 0) {
    return null;
  }

  return (
    <div className="mb-3 space-y-2" data-testid="split-run-intent-transcript">
      {messages.map((message) => (
        <TranscriptMessage key={message.id} message={message} />
      ))}
    </div>
  );
}

function TranscriptMessage({ message }: { message: CreateWithAgentMessage }) {
  if (message.role === "user") {
    const label = message.origin === "survey" ? CREATE_WITH_AGENT_COPY.youSurvey : CREATE_WITH_AGENT_COPY.you;
    return (
      <div className="flex w-full items-start">
        <span className="inline-flex w-4 shrink-0" aria-hidden />
        <div className="min-w-0 flex-1 whitespace-normal break-words rounded-md border-l-2 border-primary/50 bg-primary/10 px-2 py-1">
          <span className="mb-0.5 block font-sans text-[11px] font-medium leading-none text-primary">{label}</span>
          <MarkdownContent content={message.text} variant="workspace" className={MESSAGE_MARKDOWN} />
        </div>
      </div>
    );
  }

  return (
    <div className="flex w-full items-start">
      <span className="inline-flex w-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1 whitespace-normal break-words py-0.5 leading-5 text-foreground">
        <MarkdownContent content={message.text} variant="workspace" className={MESSAGE_MARKDOWN} />
      </div>
    </div>
  );
}
