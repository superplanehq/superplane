import { MarkdownContent } from "@/pages/app/Markdown";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMessage } from "../createWithAgentTypes";

const MESSAGE_MARKDOWN =
  "max-w-none font-sans text-[14px] leading-6 text-foreground [&_p:first-child]:mt-0 [&_p:last-child]:mb-0";

export function WorkOrderIntentTranscript({
  messages,
  streaming = false,
}: {
  messages: CreateWithAgentMessage[];
  streaming?: boolean;
}) {
  if (messages.length === 0) {
    return null;
  }

  const last = messages.at(-1);

  return (
    <div className="mb-4 space-y-4" data-testid="split-run-intent-transcript">
      {messages.map((message) => (
        <TranscriptMessage
          key={message.id}
          message={message}
          streaming={streaming && last?.role === "agent" && message.id === last.id}
        />
      ))}
    </div>
  );
}

function TranscriptMessage({ message, streaming }: { message: CreateWithAgentMessage; streaming: boolean }) {
  if (message.role === "user") {
    const isSurvey = message.origin === "survey";
    return (
      <div className="sp-text-reveal flex w-full justify-start">
        <div className="max-w-[92%] rounded-2xl bg-muted px-3.5 py-2.5">
          {isSurvey ? (
            <span className="mb-1 block font-sans text-[11px] leading-none text-muted-foreground">
              {CREATE_WITH_AGENT_COPY.youSurvey}
            </span>
          ) : null}
          <MarkdownContent content={message.text} variant="workspace" className={MESSAGE_MARKDOWN} />
        </div>
      </div>
    );
  }

  return (
    <div className="flex w-full items-start">
      <div
        className={`min-w-0 flex-1 whitespace-normal break-words text-[14px] leading-6 text-foreground ${streaming ? "sp-stream-text" : "sp-text-reveal"}`}
      >
        <MarkdownContent content={message.text} variant="workspace" className={MESSAGE_MARKDOWN} />
      </div>
    </div>
  );
}
