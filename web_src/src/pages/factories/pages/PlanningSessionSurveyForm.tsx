import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useState } from "react";

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
  const hasAnswer = answers.some((answer) => Boolean(answer?.trim()));

  if (!question) {
    return null;
  }

  return (
    <form
      className="border-b border-border bg-background px-3 py-3"
      data-testid="create-with-agent-survey"
      onSubmit={(event) => {
        event.preventDefault();
        if (!isLast) {
          setCurrentIndex((index) => index + 1);
          return;
        }
        onSubmit(formatPlanningSurveyReply(survey.questions, answers));
      }}
    >
      <div className="overflow-hidden rounded-lg border border-border">
        <div className="flex items-start justify-between gap-3 border-b border-border bg-muted/40 px-3 py-2">
          <p className="text-[12px] font-medium text-foreground">{question.prompt}</p>
          {questionCount > 1 ? (
            <span className="shrink-0 text-[10px] font-medium text-muted-foreground">
              {currentIndex + 1} of {questionCount}
            </span>
          ) : null}
        </div>
        <div className="flex flex-col gap-1 p-2">
          {question.options.map((option, optionIndex) => {
            const selected = answers[currentIndex] === option;
            return (
              <Button
                key={option}
                type="button"
                variant="ghost"
                size="sm"
                className={cn(
                  "h-auto justify-start whitespace-normal px-2.5 py-1.5 text-left text-[12px]",
                  selected
                    ? "bg-muted text-foreground ring-1 ring-border"
                    : "text-muted-foreground hover:text-foreground",
                )}
                onClick={() => {
                  setAnswers((current) => replaceAtIndex(current, currentIndex, option));
                }}
              >
                <span className="mr-2 inline-flex size-5 shrink-0 items-center justify-center rounded bg-muted text-[10px] font-semibold text-foreground">
                  {String.fromCharCode(65 + optionIndex)}
                </span>
                {option}
              </Button>
            );
          })}
          <Input
            value={customInputs[currentIndex] ?? ""}
            placeholder={CREATE_WITH_AGENT_COPY.otherAnswer}
            aria-label={`${question.prompt} ${CREATE_WITH_AGENT_COPY.otherAnswer}`}
            className="mt-0.5 h-8 text-[12px]"
            onChange={(event) => {
              const value = event.target.value;
              setCustomInputs((current) => replaceAtIndex(current, currentIndex, value));
              setAnswers((current) => replaceAtIndex(current, currentIndex, value.trim() || null));
            }}
          />
        </div>
        <div className="flex items-center justify-between gap-2 px-3 pb-3 pt-1">
          {questionCount > 1 ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 text-xs text-muted-foreground"
              disabled={isFirst}
              onClick={() => setCurrentIndex((index) => index - 1)}
            >
              <ChevronLeft size={12} className="mr-1" />
              {CREATE_WITH_AGENT_COPY.previousQuestion}
            </Button>
          ) : (
            <span />
          )}
          {isLast ? (
            <Button type="submit" size="sm" className="h-7 text-xs">
              {hasAnswer ? CREATE_WITH_AGENT_COPY.sendAnswers : CREATE_WITH_AGENT_COPY.skipSurvey}
            </Button>
          ) : (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 text-xs text-muted-foreground"
              onClick={() => setCurrentIndex((index) => index + 1)}
            >
              {CREATE_WITH_AGENT_COPY.nextQuestion}
              <ChevronRight size={12} className="ml-1" />
            </Button>
          )}
        </div>
      </div>
    </form>
  );
}

function replaceAtIndex<T>(items: T[], index: number, value: T): T[] {
  const next = [...items];
  next[index] = value;
  return next;
}
