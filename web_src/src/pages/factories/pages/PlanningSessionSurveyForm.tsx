import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useState, type KeyboardEvent } from "react";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentSurvey } from "./createWithAgentTypes";
import { formatPlanningSurveyReply } from "./planningSessionSurvey";

export function PlanningSessionSurveyForm({
  survey,
  onSubmit,
}: {
  survey: CreateWithAgentSurvey;
  onSubmit: (text: string) => void;
}) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<Array<string | null>>(() => survey.questions.map(() => null));
  const [customInputs, setCustomInputs] = useState<string[]>(() => survey.questions.map(() => ""));
  const question = survey.questions[currentIndex];
  const questionCount = survey.questions.length;
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === questionCount - 1;
  const hasCurrentAnswer = Boolean(answers[currentIndex]?.trim());
  const sendReply = () => onSubmit(formatPlanningSurveyReply(survey.questions, answers));

  if (!question) {
    return null;
  }

  return (
    <div className="sp-survey-enter mt-3 mb-1" data-testid="create-with-agent-survey">
      <div className="rounded-2xl border bg-card px-3.5 py-3 shadow-xs" data-testid="create-with-agent-survey-card">
        <div className="flex items-center justify-between gap-3 text-[11px] leading-4 text-muted-foreground">
          <p className="font-medium">{CREATE_WITH_AGENT_COPY.surveyHeader}</p>
          {questionCount > 1 ? (
            <span className="shrink-0 tabular-nums">
              {currentIndex + 1} of {questionCount}
            </span>
          ) : null}
        </div>
        <div key={question.prompt} className="sp-survey-page">
          <p className="mt-2 text-[14px] font-medium leading-5 text-foreground">{question.prompt}</p>
          <div className="mt-3 flex flex-col gap-1.5">
            {question.options.map((option, optionIndex) => {
              const selected = answers[currentIndex] === option;
              const keyLabel = String.fromCharCode(65 + optionIndex);
              return (
                <Button
                  key={option}
                  type="button"
                  variant="ghost"
                  size="sm"
                  aria-pressed={selected}
                  className={cn(
                    "h-auto justify-start gap-2.5 whitespace-normal rounded-xl border px-2.5 py-2 text-left text-[13px] leading-5 shadow-none",
                    selected
                      ? "border-primary bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground"
                      : "border-border bg-background text-foreground hover:bg-muted",
                  )}
                  onClick={() => {
                    setAnswers((current) => replaceAtIndex(current, currentIndex, option));
                  }}
                >
                  <span
                    className={cn(
                      "inline-flex size-5 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold",
                      selected ? "bg-primary-foreground/15 text-primary-foreground" : "bg-muted text-muted-foreground",
                    )}
                    aria-hidden
                  >
                    {selected ? <SurveyOptionCheck /> : keyLabel}
                  </span>
                  {option}
                </Button>
              );
            })}
            <Input
              type="text"
              value={customInputs[currentIndex] ?? ""}
              placeholder={CREATE_WITH_AGENT_COPY.otherAnswer}
              aria-label={`${question.prompt} ${CREATE_WITH_AGENT_COPY.otherAnswer}`}
              className="mt-0.5 h-9 rounded-xl border-dashed bg-background text-[13px] shadow-none"
              onChange={(event) => {
                const value = event.target.value;
                setCustomInputs((current) => replaceAtIndex(current, currentIndex, value));
                setAnswers((current) => replaceAtIndex(current, currentIndex, value.trim() || null));
              }}
              onKeyDown={(event) =>
                handleSurveyEnter(
                  event,
                  isLast,
                  hasCurrentAnswer,
                  () => setCurrentIndex((index) => index + 1),
                  sendReply,
                )
              }
            />
          </div>
        </div>
        <SurveyFormPager
          questionCount={questionCount}
          isFirst={isFirst}
          isLast={isLast}
          hasCurrentAnswer={hasCurrentAnswer}
          onPrevious={() => setCurrentIndex((index) => index - 1)}
          onNext={() => setCurrentIndex((index) => index + 1)}
          onSend={sendReply}
        />
      </div>
    </div>
  );
}

function SurveyOptionCheck() {
  return (
    <svg viewBox="0 0 16 16" className="size-3.5" aria-hidden>
      <path
        className="sp-survey-check"
        d="M3.5 8.5 L6.5 11.5 L12.5 4.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function handleSurveyEnter(
  event: KeyboardEvent<HTMLInputElement>,
  isLast: boolean,
  hasCurrentAnswer: boolean,
  onNext: () => void,
  onSend: () => void,
) {
  if (event.key !== "Enter") {
    return;
  }
  event.preventDefault();
  if (!isLast) {
    onNext();
    return;
  }
  if (hasCurrentAnswer) {
    onSend();
  }
}

function SurveyFormPager({
  questionCount,
  isFirst,
  isLast,
  hasCurrentAnswer,
  onPrevious,
  onNext,
  onSend,
}: {
  questionCount: number;
  isFirst: boolean;
  isLast: boolean;
  hasCurrentAnswer: boolean;
  onPrevious: () => void;
  onNext: () => void;
  onSend: () => void;
}) {
  return (
    <div className="mt-2 flex items-center justify-between gap-2">
      {questionCount > 1 ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-8 text-[13px] text-muted-foreground"
          disabled={isFirst}
          onClick={onPrevious}
        >
          <ChevronLeft size={12} className="mr-1" />
          {CREATE_WITH_AGENT_COPY.previousQuestion}
        </Button>
      ) : (
        <span />
      )}
      {isLast ? (
        <div className="flex items-center gap-2">
          {questionCount > 1 ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-8 text-[13px] text-muted-foreground"
              onClick={onSend}
            >
              {CREATE_WITH_AGENT_COPY.skipSurvey}
            </Button>
          ) : null}
          <Button
            type="button"
            size="sm"
            className="h-8 text-[13px]"
            disabled={questionCount > 1 && !hasCurrentAnswer}
            onClick={onSend}
          >
            {hasCurrentAnswer || questionCount > 1
              ? CREATE_WITH_AGENT_COPY.sendAnswers
              : CREATE_WITH_AGENT_COPY.skipSurvey}
          </Button>
        </div>
      ) : (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-8 text-[13px] text-muted-foreground"
          onClick={onNext}
        >
          {CREATE_WITH_AGENT_COPY.nextQuestion}
          <ChevronRight size={12} className="ml-1" />
        </Button>
      )}
    </div>
  );
}

function replaceAtIndex<T>(items: T[], index: number, value: T): T[] {
  const next = [...items];
  next[index] = value;
  return next;
}
