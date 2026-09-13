import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMessage } from "../createWithAgentTypes";
import { parsePlanningSurveyReply } from "../planningSessionSurvey";

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
    if (message.origin === "survey") {
      return <SurveyAnswerBubble text={message.text} />;
    }
    return (
      <div className="sp-text-reveal flex w-full justify-start">
        <div
          className="sp-user-note max-w-[92%] rounded-2xl border px-3.5 py-2.5"
          data-testid="split-run-intent-user-note"
        >
          <p className="sp-user-note-label mb-1.5 text-[11px] font-medium leading-none">{CREATE_WITH_AGENT_COPY.you}</p>
          <WorkOrderDescription
            description={message.text}
            previewHeight={FALLBACK_COLLAPSED_MAX_HEIGHT_PX}
            fadeClassName="sp-user-note-fade"
          />
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

function SurveyAnswerBubble({ text }: { text: string }) {
  const pairs = parsePlanningSurveyReply(text);
  const skipped = text.trim() === CREATE_WITH_AGENT_COPY.surveySkipped;

  return (
    <div className="sp-text-reveal flex w-full justify-start">
      <div
        className="sp-survey-card max-w-[92%] rounded-2xl border px-3.5 py-3"
        data-testid="split-run-intent-survey-answer"
      >
        <p className="sp-survey-accent text-[11px] font-medium leading-none">{CREATE_WITH_AGENT_COPY.youSurvey}</p>
        {pairs.length > 0 ? (
          <ul className={cn("mt-2.5", pairs.length > 1 ? "divide-y divide-border" : "space-y-2.5")}>
            {pairs.map((pair) => (
              <li key={pair.question} className={cn("space-y-1", pairs.length > 1 && "py-2.5 first:pt-0 last:pb-0")}>
                <p className="text-[12px] leading-4 text-muted-foreground">{pair.question}</p>
                <p
                  className={cn(
                    "text-[14px] leading-5 font-medium text-foreground",
                    pair.answer === "skipped" && "font-normal text-muted-foreground",
                  )}
                >
                  {pair.answer === "skipped" ? CREATE_WITH_AGENT_COPY.surveyAnswerSkipped : pair.answer}
                </p>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-2.5 text-[14px] leading-5 text-foreground">
            {skipped ? CREATE_WITH_AGENT_COPY.surveySkipped : text}
          </p>
        )}
      </div>
    </div>
  );
}
