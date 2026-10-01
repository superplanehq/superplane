import { memo, useEffect, useLayoutEffect, useRef } from "react";

import type { FilesFile } from "@/api-client";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import type { OrgUserDisplay, OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { streamWordsIn } from "@/lib/streamWords";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import { OrgUserReference } from "../../OrgUserReference";
import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMessage } from "../createWithAgentTypes";
import { parsePlanningSurveyReply } from "../planningSessionSurvey";
import { AgentActivityView } from "./AgentActivityView";
import type { AgentActivity } from "./agentActivity";
import {
  SENDER_ROW_CLASSNAME,
  SURVEY_PICK_CLASSNAME,
  SURVEY_SKIPPED_CLASSNAME,
  USER_BUBBLE_CLASSNAME,
  USER_BUBBLE_FADE_CLASSNAME,
  USER_TURN_CLASSNAME,
} from "./chatBubbleStyle";

const MESSAGE_MARKDOWN = "max-w-none font-sans text-[14px] leading-5 text-foreground [&_p]:!my-0";

export function WorkOrderIntentTranscript({
  messages,
  organizationId,
  streaming = false,
  files,
  activities = [],
}: {
  messages: CreateWithAgentMessage[];
  organizationId: string;
  streaming?: boolean;
  files?: FilesFile[];
  activities?: AgentActivity[];
}) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const visible = messages.filter((message) => message.kind !== "plan");
  const knownMessageIDs = useRef<Set<string> | null>(null);
  const newAgentMessageIDs = new Set(
    knownMessageIDs.current
      ? visible
          .filter((message) => message.role === "agent" && !knownMessageIDs.current?.has(message.id))
          .map((message) => message.id)
      : [],
  );

  useLayoutEffect(() => {
    knownMessageIDs.current = new Set(messages.map((message) => message.id));
  }, [messages]);

  if (visible.length === 0 && activities.length === 0) {
    return null;
  }

  const last = visible.at(-1);
  const activitiesByID = new Map(activities.map((activity) => [activity.id, activity]));
  const linkedActivityIDs = new Set(visible.flatMap((message) => (message.activityId ? [message.activityId] : [])));
  const unlinkedActivities = activities.filter(
    (activity) => activity.status !== "running" && !linkedActivityIDs.has(activity.id),
  );

  return (
    <div className="space-y-0" data-testid="split-run-intent-transcript">
      {visible.map((message, index) => {
        const activity = message.activityId ? activitiesByID.get(message.activityId) : undefined;
        return (
          <div key={message.id}>
            {activity ? <AgentActivityView activity={activity} /> : null}
            <TranscriptMessage
              message={message}
              resolveUser={resolveUser}
              streaming={
                newAgentMessageIDs.has(message.id) || (streaming && last?.role === "agent" && message.id === last.id)
              }
              files={files}
              adjacentUserAbove={visible[index - 1]?.role === "user"}
              adjacentUserBelow={visible[index + 1]?.role === "user"}
              sameSenderAbove={sameUserSender(visible[index - 1], message)}
            />
          </div>
        );
      })}
      {unlinkedActivities.map((activity) => (
        <AgentActivityView key={activity.id} activity={activity} />
      ))}
    </div>
  );
}

function TranscriptMessage({
  message,
  resolveUser,
  streaming,
  files,
  adjacentUserAbove,
  adjacentUserBelow,
  sameSenderAbove,
}: {
  message: CreateWithAgentMessage;
  resolveUser: OrgUserDisplayLookup;
  streaming: boolean;
  files?: FilesFile[];
  adjacentUserAbove: boolean;
  adjacentUserBelow: boolean;
  sameSenderAbove: boolean;
}) {
  if (message.kind === "plan") {
    return null;
  }
  if (message.role === "user") {
    const frameClassName = userMessageFrameClass(adjacentUserAbove, adjacentUserBelow);
    const sender = sameSenderAbove ? null : senderDisplay(message.userId, resolveUser);
    if (message.origin === "survey") {
      return <SurveyAnswerBubble text={message.text} sender={sender} frameClassName={frameClassName} />;
    }
    return <ComposerNoteBubble text={message.text} sender={sender} files={files} frameClassName={frameClassName} />;
  }

  return <AgentMessage text={message.text} streaming={streaming} files={files} />;
}

function userMessageFrameClass(adjacentUserAbove: boolean, adjacentUserBelow: boolean) {
  return cn(
    "sp-text-reveal flex w-full justify-end",
    adjacentUserAbove ? "pt-1.5" : "pt-2.5",
    adjacentUserBelow ? "pb-1.5" : "pb-2.5",
  );
}

/** Consecutive turns from one person show the sender once, like a grouped bubble. */
function sameUserSender(previous: CreateWithAgentMessage | undefined, current: CreateWithAgentMessage): boolean {
  if (!previous || previous.role !== "user" || current.role !== "user") {
    return false;
  }
  const previousId = previous.userId?.trim();
  return Boolean(previousId) && previousId === current.userId?.trim();
}

const AgentMessage = memo(function AgentMessage({
  text,
  streaming,
  files,
}: {
  text: string;
  streaming: boolean;
  files?: FilesFile[];
}) {
  const contentRef = useRef<HTMLDivElement>(null);
  const animateWords = useRef(streaming).current;

  useEffect(() => {
    if (!animateWords || !contentRef.current) return;
    return streamWordsIn(contentRef.current);
  }, [animateWords]);

  return (
    <div className="flex w-full items-start px-2 py-0.5" data-testid="split-run-intent-agent-message">
      <div
        ref={contentRef}
        className={`min-w-0 flex-1 whitespace-normal break-words text-[14px] leading-5 text-foreground ${animateWords ? "sp-stream-words" : "sp-text-reveal"}`}
      >
        <MarkdownContent content={text} files={files} variant="workspace" className={MESSAGE_MARKDOWN} />
      </div>
    </div>
  );
}, sameAgentMessage);

function sameAgentMessage(
  previous: { text: string; streaming: boolean; files?: FilesFile[] },
  next: { text: string; streaming: boolean; files?: FilesFile[] },
): boolean {
  return previous.text === next.text && previous.files === next.files;
}

function ComposerNoteBubble({
  text,
  sender,
  files,
  frameClassName,
}: {
  text: string;
  sender: OrgUserDisplay | null;
  files?: FilesFile[];
  frameClassName: string;
}) {
  return (
    <div className={frameClassName}>
      <div className={USER_TURN_CLASSNAME} data-testid="split-run-intent-user-note">
        {sender ? <ChatSenderRow display={sender} /> : null}
        <div className={USER_BUBBLE_CLASSNAME}>
          <WorkOrderDescription
            description={text}
            files={files}
            previewHeight={FALLBACK_COLLAPSED_MAX_HEIGHT_PX}
            fadeClassName={USER_BUBBLE_FADE_CLASSNAME}
          />
        </div>
      </div>
    </div>
  );
}

/** Survey answers read as quick-reply picks: the question in small muted text, the pick as a primary bubble. */
function SurveyAnswerBubble({
  text,
  sender,
  frameClassName,
}: {
  text: string;
  sender: OrgUserDisplay | null;
  frameClassName: string;
}) {
  const pairs = parsePlanningSurveyReply(text);
  const skipped = text.trim() === CREATE_WITH_AGENT_COPY.surveySkipped;

  return (
    <div className={frameClassName}>
      <div className={USER_TURN_CLASSNAME} data-testid="split-run-intent-survey-answer">
        {sender ? <ChatSenderRow display={sender} prefix={CREATE_WITH_AGENT_COPY.answeredBy} /> : null}
        {pairs.length > 0 ? (
          <ul className="flex w-full flex-col items-end gap-2">
            {pairs.map((pair) => (
              <li key={pair.question} className="flex max-w-full flex-col items-end gap-1">
                <p className="px-1 text-right text-[12px] leading-4 text-muted-foreground">{pair.question}</p>
                {pair.answer === "skipped" ? (
                  <span className={SURVEY_SKIPPED_CLASSNAME}>{CREATE_WITH_AGENT_COPY.surveyAnswerSkipped}</span>
                ) : (
                  <span className={SURVEY_PICK_CLASSNAME} data-testid="split-run-intent-survey-pick">
                    {pair.answer}
                  </span>
                )}
              </li>
            ))}
          </ul>
        ) : skipped ? (
          <span className={SURVEY_SKIPPED_CLASSNAME}>{CREATE_WITH_AGENT_COPY.surveySkipped}</span>
        ) : (
          <span className={SURVEY_PICK_CLASSNAME} data-testid="split-run-intent-survey-pick">
            {text}
          </span>
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

/** Sender row above an outgoing turn: optional prefix, avatar, given name. */
function ChatSenderRow({ display, prefix }: { display: OrgUserDisplay; prefix?: string }) {
  return (
    <div className={SENDER_ROW_CLASSNAME}>
      {prefix ? <span className="shrink-0">{prefix}</span> : null}
      <span className="inline-flex min-w-0 items-center gap-1.5" title={display.name}>
        <span className="inline-flex size-5 shrink-0 items-center justify-center">
          <OrgUserReference display={display} size="xs" showName={false} className="rounded-full leading-none" />
        </span>
        <span className="truncate">{senderGivenName(display.name)}</span>
      </span>
    </div>
  );
}

function senderGivenName(fullName: string): string {
  const givenName = fullName.trim().split(/\s+/)[0];
  return givenName || fullName;
}
