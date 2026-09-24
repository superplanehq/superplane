import { CheckCircle2, TriangleAlert } from "lucide-react";

import githubIcon from "@/assets/icons/integrations/github.svg";
import { cn } from "@/lib/utils";

import { CONFIDENCE_SCORE_MAX } from "../lib/confidenceScore";
import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import {
  planningSetupPreviewCaption,
  planningSetupPreviewScene,
  type PlanningSetupPreviewChat,
  type PlanningSetupPreviewScene,
  type PlanningSetupPreviewScore,
  type PlanningSetupStep,
} from "./planningSetupCaption";
import { PRFeedbackSetupPreviewPane } from "./PRFeedbackSetupWizardChrome";

const PANE_LABEL_CLASS = "text-[11px] font-medium uppercase tracking-[0.12em] text-muted-foreground";

const SCORE_NAME: Record<PlanningSetupPreviewScore["kind"], string> = {
  clarity: PLANNING_SETTINGS_COPY.wizardPreviewClarityName,
  confidence: PLANNING_SETTINGS_COPY.wizardPreviewConfidenceName,
};

type ChatScript = {
  message: string;
  question: { text: string; options: readonly [string, string] };
};

const CHAT_SCRIPTS: Record<PlanningSetupPreviewChat, ChatScript> = {
  question: {
    message: PLANNING_SETTINGS_COPY.wizardPreviewAgentMessage,
    question: {
      text: PLANNING_SETTINGS_COPY.wizardPreviewAgentQuestion,
      options: [PLANNING_SETTINGS_COPY.wizardPreviewOptionOne, PLANNING_SETTINGS_COPY.wizardPreviewOptionTwo],
    },
  },
  confidence: {
    message: PLANNING_SETTINGS_COPY.wizardPreviewConfidenceMessage,
    question: {
      text: PLANNING_SETTINGS_COPY.wizardPreviewConfidenceQuestion,
      options: [
        PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOptionOne,
        PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOptionTwo,
      ],
    },
  },
  clarity: {
    message: PLANNING_SETTINGS_COPY.wizardPreviewClarityMessage,
    question: {
      text: PLANNING_SETTINGS_COPY.wizardPreviewClarityQuestion,
      options: [
        PLANNING_SETTINGS_COPY.wizardPreviewClarityOptionOne,
        PLANNING_SETTINGS_COPY.wizardPreviewClarityOptionTwo,
      ],
    },
  },
};

export function PlanningSetupPreview({
  step,
  enabled,
  clarity,
  confidence,
}: {
  step: PlanningSetupStep;
  enabled: boolean;
  clarity: boolean;
  confidence: boolean;
}) {
  const input = { step, enabled, clarity, confidence };
  const caption = planningSetupPreviewCaption(input);
  const scene = planningSetupPreviewScene(input);

  return (
    <PRFeedbackSetupPreviewPane
      label={PLANNING_SETTINGS_COPY.wizardPreviewLabel}
      caption={caption}
      testId="planning-setup-preview"
      captionTestId="planning-setup-preview-caption"
    >
      <article
        className="w-full max-w-md overflow-hidden rounded-xl border border-border bg-card shadow-sm"
        data-testid="planning-setup-preview-card"
      >
        <DraftHeader />
        <div
          key={`${step}-${enabled}-${clarity}-${confidence}`}
          className="motion-safe:animate-in motion-safe:fade-in motion-safe:duration-200"
        >
          {scene.body === "source" ? <SourceBody /> : <PlanBody scene={scene} />}
          <DraftFooter scene={scene} />
        </div>
      </article>
    </PRFeedbackSetupPreviewPane>
  );
}

function DraftHeader() {
  return (
    <header className="flex items-center justify-between gap-3 border-b border-border px-4 py-2.5">
      <p className="truncate text-[13px] font-medium text-foreground" data-testid="planning-setup-preview-title">
        {PLANNING_SETTINGS_COPY.wizardPreviewDraftTitle}
      </p>
      <div className="flex items-center gap-1" aria-hidden>
        <span className="size-1.5 rounded-full bg-muted-foreground/40" />
        <span className="size-1.5 rounded-full bg-muted-foreground/40" />
        <span className="size-1.5 rounded-full bg-muted-foreground/40" />
      </div>
    </header>
  );
}

function SourceBody() {
  return (
    <div className="grid min-h-[280px] grid-cols-[40%_1fr]" data-testid="planning-setup-preview-source">
      <div className="space-y-4 border-r border-border px-4 py-4">
        <div className="space-y-1.5">
          <p className={PANE_LABEL_CLASS}>{PLANNING_SETTINGS_COPY.wizardPreviewSource}</p>
          <p className="flex items-center gap-1.5 text-[12px] text-foreground">
            <img
              src={githubIcon}
              alt={PLANNING_SETTINGS_COPY.wizardPreviewSourceName}
              className="size-3.5 shrink-0 dark:brightness-0 dark:invert"
            />
            <span>{PLANNING_SETTINGS_COPY.wizardPreviewSourceName}</span>
          </p>
          <p className="text-[12px] font-medium text-foreground" data-testid="planning-setup-preview-issue">
            {PLANNING_SETTINGS_COPY.wizardPreviewIssue}
          </p>
        </div>
        <div className="space-y-1.5">
          <p className={PANE_LABEL_CLASS}>{PLANNING_SETTINGS_COPY.wizardPreviewArtifacts}</p>
          <p className="text-[12px] text-muted-foreground">{PLANNING_SETTINGS_COPY.wizardPreviewArtifactsNone}</p>
        </div>
      </div>
      <div className="px-4 py-4">
        <p className="text-[13px] leading-5 text-foreground">{PLANNING_SETTINGS_COPY.wizardPreviewSourceBody}</p>
      </div>
    </div>
  );
}

function PlanBody({ scene }: { scene: PlanningSetupPreviewScene }) {
  return (
    <div className="grid min-h-[280px] grid-cols-[40%_1fr]" data-testid="planning-setup-preview-plan">
      <div className="space-y-2 border-r border-border px-4 py-4">
        <p className={PANE_LABEL_CLASS}>{PLANNING_SETTINGS_COPY.wizardPreviewChat}</p>
        <PreviewBubble align="end">{PLANNING_SETTINGS_COPY.wizardPreviewUserMessage}</PreviewBubble>
        <AgentChat chat={scene.chat} />
      </div>
      <div className="flex flex-col gap-3 px-4 py-4">
        <p className={PANE_LABEL_CLASS}>{PLANNING_SETTINGS_COPY.wizardPreviewPlan}</p>
        <ol className="list-decimal space-y-1 pl-4 text-[12px] leading-5 text-foreground">
          {PLANNING_SETTINGS_COPY.wizardPreviewPlanSteps.map((stepText) => (
            <li key={stepText}>{stepText}</li>
          ))}
        </ol>
        <ScoreRows scores={scene.scores} />
      </div>
    </div>
  );
}

function AgentChat({ chat }: { chat: PlanningSetupPreviewChat }) {
  const script = CHAT_SCRIPTS[chat];
  return (
    <div className="space-y-2" data-testid="planning-setup-preview-chat" data-chat={chat}>
      <PreviewBubble align="start">{script.message}</PreviewBubble>
      <div className="space-y-2" data-testid="planning-setup-preview-question">
        <PreviewBubble align="start">{script.question.text}</PreviewBubble>
        <div className="flex flex-wrap gap-1.5">
          <OptionPill selected>{script.question.options[0]}</OptionPill>
          <OptionPill>{script.question.options[1]}</OptionPill>
        </div>
      </div>
    </div>
  );
}

function OptionPill({ selected = false, children }: { selected?: boolean; children: string }) {
  return (
    <span
      className={cn(
        "rounded-full border px-2 py-0.5 text-[11px]",
        selected ? "border-foreground bg-foreground text-background" : "border-border text-muted-foreground",
      )}
    >
      {children}
    </span>
  );
}

function ScoreRows({ scores }: { scores: PlanningSetupPreviewScore[] }) {
  if (scores.length === 0) {
    return null;
  }
  return (
    <div className="mt-auto space-y-2 border-t border-border pt-3" data-testid="planning-setup-preview-scores">
      {scores.map((score) => (
        <ScoreRow key={score.kind} score={score} />
      ))}
    </div>
  );
}

function ScoreRow({ score }: { score: PlanningSetupPreviewScore }) {
  const label = `${SCORE_NAME[score.kind]} ${score.score}/${CONFIDENCE_SCORE_MAX}`;
  return (
    <div
      className={cn("space-y-1", score.emphasized ? "text-foreground" : "text-muted-foreground/70")}
      data-testid={`planning-setup-preview-${score.kind}`}
      data-emphasized={score.emphasized ? "true" : "false"}
    >
      <div className="flex items-center gap-2 text-[12px]">
        <span className={score.emphasized ? "font-medium" : undefined}>{label}</span>
        <ScoreBars score={score.score} emphasized={score.emphasized} />
      </div>
      {score.summary ? <p className="text-[12px] leading-5 text-muted-foreground">{score.summary}</p> : null}
    </div>
  );
}

function ScoreBars({ score, emphasized }: { score: number; emphasized: boolean }) {
  return (
    <span className="flex items-center gap-0.5" aria-hidden>
      {Array.from({ length: CONFIDENCE_SCORE_MAX }, (_, index) => (
        <span
          key={index}
          className={cn(
            "h-2.5 w-1 rounded-sm",
            index < score ? (emphasized ? "bg-foreground" : "bg-muted-foreground/40") : "bg-border",
          )}
        />
      ))}
    </span>
  );
}

function DraftFooter({ scene }: { scene: PlanningSetupPreviewScene }) {
  const caution = scene.note === "caution";
  const Icon = caution ? TriangleAlert : CheckCircle2;
  return (
    <footer
      className={cn(
        "flex items-center justify-between gap-3 border-t px-4 py-2.5",
        caution
          ? "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)]"
          : "border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)]",
      )}
      data-testid="planning-setup-preview-note"
      data-tone={scene.note}
    >
      <div className="flex min-w-0 items-start gap-2">
        <Icon
          className={cn(
            "mt-0.5 size-3.5 shrink-0",
            caution ? "text-[color:var(--status-waiting-fg)]" : "text-[color:var(--status-completed-fg)]",
          )}
          aria-hidden
        />
        <div className="min-w-0">
          <p className="text-[12px] font-medium text-foreground">
            {caution ? PLANNING_SETTINGS_COPY.wizardPreviewCautionHeadline : PLANNING_SETTINGS_COPY.wizardPreviewReady}
          </p>
          {caution ? (
            <p className="text-[11px] leading-4 text-muted-foreground">
              {PLANNING_SETTINGS_COPY.wizardPreviewCautionText}
            </p>
          ) : null}
        </div>
      </div>
      {scene.showStart ? (
        <span
          className="shrink-0 rounded-md bg-foreground px-2 py-1 text-[11px] font-medium text-background"
          data-testid="planning-setup-preview-start"
        >
          {PLANNING_SETTINGS_COPY.wizardPreviewStart}
        </span>
      ) : null}
    </footer>
  );
}

function PreviewBubble({ align, children }: { align: "start" | "end"; children: string }) {
  return (
    <p
      className={
        align === "end"
          ? "ml-4 rounded-lg bg-foreground px-2.5 py-1.5 text-[12px] leading-5 text-background"
          : "mr-4 rounded-lg bg-muted px-2.5 py-1.5 text-[12px] leading-5 text-foreground"
      }
    >
      {children}
    </p>
  );
}
