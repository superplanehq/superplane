import type { FilesFile } from "@/api-client";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import type { OrgUserDisplay, OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import { OrgUserReference } from "../../OrgUserReference";
import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMessage } from "../createWithAgentTypes";
import { parsePlanningSurveyReply } from "../planningSessionSurvey";

const MESSAGE_MARKDOWN =
  "max-w-none font-sans text-[14px] leading-6 text-foreground [&_p:first-child]:mt-0 [&_p:last-child]:mb-0";

export function WorkOrderIntentTranscript({
  messages,
  organizationId,
  streaming = false,
  files,
}: {
  messages: CreateWithAgentMessage[];
  organizationId: string;
  streaming?: boolean;
  files?: FilesFile[];
}) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const visible = messages.filter((message) => message.kind !== "plan");

  if (visible.length === 0) {
    return null;
  }

  const last = visible.at(-1);

  return (
    <div className="mb-4 space-y-4" data-testid="split-run-intent-transcript">
      {visible.map((message) => (
        <TranscriptMessage
          key={message.id}
          message={message}
          resolveUser={resolveUser}
          streaming={streaming && last?.role === "agent" && message.id === last.id}
          files={files}
        />
      ))}
    </div>
  );
}

function TranscriptMessage({
  message,
  resolveUser,
  streaming,
  files,
}: {
  message: CreateWithAgentMessage;
  resolveUser: OrgUserDisplayLookup;
  streaming: boolean;
  files?: FilesFile[];
}) {
  if (message.kind === "plan") {
    return null;
  }
  if (message.role === "user") {
    if (message.origin === "survey") {
      return <SurveyAnswerBubble text={message.text} userId={message.userId} resolveUser={resolveUser} />;
    }
    return <ComposerNoteBubble text={message.text} userId={message.userId} resolveUser={resolveUser} files={files} />;
  }

  return (
    <div className="flex w-full items-start">
      <div
        className={`min-w-0 flex-1 whitespace-normal break-words text-[14px] leading-6 text-foreground ${streaming ? "sp-stream-text" : "sp-text-reveal"}`}
      >
        <MarkdownContent content={message.text} files={files} variant="workspace" className={MESSAGE_MARKDOWN} />
      </div>
    </div>
  );
}

function ComposerNoteBubble({
  text,
  userId,
  resolveUser,
  files,
}: {
  text: string;
  userId?: string;
  resolveUser: OrgUserDisplayLookup;
  files?: FilesFile[];
}) {
  const sender = senderDisplay(userId, resolveUser);

  return (
    <div className="sp-text-reveal flex w-full justify-end">
      <div
        className="sp-user-note max-w-[92%] rounded-2xl border px-3.5 py-2.5"
        data-testid="split-run-intent-user-note"
      >
        {sender ? (
          <div className="sp-user-note-label mb-1.5">
            <ChatSenderMark display={sender} />
          </div>
        ) : null}
        <WorkOrderDescription
          description={text}
          files={files}
          previewHeight={FALLBACK_COLLAPSED_MAX_HEIGHT_PX}
          fadeClassName="sp-user-note-fade"
        />
      </div>
    </div>
  );
}

function SurveyAnswerBubble({
  text,
  userId,
  resolveUser,
}: {
  text: string;
  userId?: string;
  resolveUser: OrgUserDisplayLookup;
}) {
  const pairs = parsePlanningSurveyReply(text);
  const skipped = text.trim() === CREATE_WITH_AGENT_COPY.surveySkipped;
  const sender = senderDisplay(userId, resolveUser);

  return (
    <div className="sp-text-reveal flex w-full justify-end">
      <div
        className="sp-survey-card max-w-[92%] rounded-2xl border px-3.5 py-3"
        data-testid="split-run-intent-survey-answer"
      >
        {sender ? (
          <div className="sp-survey-accent flex min-w-0 items-center gap-1.5 text-[11px] font-medium leading-none">
            <span>{CREATE_WITH_AGENT_COPY.answeredBy}</span>
            <ChatSenderMark display={sender} />
          </div>
        ) : null}
        {pairs.length > 0 ? (
          <ul className={cn(sender && "mt-2.5", pairs.length > 1 ? "divide-y divide-border" : "space-y-2.5")}>
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
          <p className={cn("text-[14px] leading-5 text-foreground", sender && "mt-2.5")}>
            {skipped ? CREATE_WITH_AGENT_COPY.surveySkipped : text}
          </p>
        )}
      </div>
    </div>
  );
}

function senderDisplay(userId: string | undefined, resolveUser: OrgUserDisplayLookup): OrgUserDisplay | null {
  const id = userId?.trim();
  if (!id) {
    return null;
  }
  const display = resolveUser(id);
  return display?.name?.trim() ? display : null;
}

function ChatSenderMark({ display }: { display: OrgUserDisplay }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1.5" title={display.name}>
      <span className="inline-flex size-5 shrink-0 items-center justify-center">
        <OrgUserReference display={display} size="xs" showName={false} className="rounded-full leading-none" />
      </span>
      <span className="truncate text-[11px] leading-4 text-muted-foreground">{senderGivenName(display.name)}</span>
    </span>
  );
}

function senderGivenName(fullName: string): string {
  const givenName = fullName.trim().split(/\s+/)[0];
  return givenName || fullName;
}
