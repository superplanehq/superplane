import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { ChevronLeft, ChevronRight, MessageCircleQuestion } from "lucide-react";
import { useCallback, useState, type ChangeEvent } from "react";

import { WORK_ORDER_ATTENTION_LABEL } from "./lib/workOrderAttention";
import type { WorkOrderSurveyAnswerInput, WorkOrderSurveyView } from "./lib/workOrderSurvey";

export const WORK_ORDER_SURVEY_TITLE = WORK_ORDER_ATTENTION_LABEL.question;
export const WORK_ORDER_SURVEY_HELP =
  "The run waits until you submit. If you do not answer in time, the agent continues without an answer.";
export const WORK_ORDER_SURVEY_SUBMIT = "Continue";
export const WORK_ORDER_SURVEY_CUSTOM_PLACEHOLDER = "Type your own answer";

export function WorkOrderSurveyCard({
  survey,
  canSubmit = true,
  busy = false,
  help = WORK_ORDER_SURVEY_HELP,
  onSubmit,
}: {
  survey: WorkOrderSurveyView;
  canSubmit?: boolean;
  busy?: boolean;
  help?: string;
  onSubmit?: (answers: WorkOrderSurveyAnswerInput[]) => void | Promise<void>;
}) {
  const questions = survey.questions;
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<(string | null)[]>(() => questions.map(() => null));
  const [customInputs, setCustomInputs] = useState<string[]>(() => questions.map(() => ""));

  const current = questions[currentIndex];
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === questions.length - 1;
  const progress = questions.length === 0 ? 0 : ((currentIndex + 1) / questions.length) * 100;

  const updateAnswer = useCallback(
    (value: string | null) => {
      setAnswers((currentAnswers) => replaceAtIndex(currentAnswers, currentIndex, value));
    },
    [currentIndex],
  );

  const selectOption = useCallback(
    (option: string) => {
      updateAnswer(option);
    },
    [updateAnswer],
  );

  const handleCustomInputChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.target.value;
      setCustomInputs((currentInputs) => replaceAtIndex(currentInputs, currentIndex, value));
      updateAnswer(value.trim() ? value.trim() : null);
    },
    [currentIndex, updateAnswer],
  );

  const handleSubmit = useCallback(() => {
    if (!canSubmit || busy) {
      return;
    }
    const payload = questions.map((question, index) => ({
      id: question.id,
      value: answers[index]?.trim() || "skipped",
    }));
    void onSubmit?.(payload);
  }, [answers, busy, canSubmit, onSubmit, questions]);

  if (!current) {
    return null;
  }

  return (
    <section
      className="overflow-hidden rounded-lg border bg-card"
      data-testid="work-order-survey-card"
      aria-label={WORK_ORDER_SURVEY_TITLE}
    >
      <div className="h-1 bg-muted">
        <div
          className="h-full bg-[color:var(--status-waiting-dot)] transition-[width] duration-200"
          style={{ width: `${progress}%` }}
        />
      </div>

      <div className="flex items-center gap-3 px-4 pt-4">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-blue-500/10" aria-hidden>
          <MessageCircleQuestion className="size-4 text-blue-700 dark:text-blue-400" />
        </span>
        <p className="min-w-0 flex-1 text-xs leading-snug text-muted-foreground">{help}</p>
        <span className="shrink-0 text-[11px] font-medium text-muted-foreground">
          {currentIndex + 1} of {questions.length}
        </span>
      </div>

      <div className="px-4 pt-4">
        <h2 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">{current.prompt}</h2>
      </div>

      <SurveyQuestionChoices
        options={current.options}
        selected={answers[currentIndex]}
        customValue={customInputs[currentIndex]}
        allowFreeText={current.allowFreeText}
        onSelect={selectOption}
        onCustomChange={handleCustomInputChange}
      />
      <SurveyCardNav
        isFirst={isFirst}
        isLast={isLast}
        canSubmit={canSubmit}
        busy={busy}
        onPrevious={() => setCurrentIndex((index) => index - 1)}
        onNext={() => setCurrentIndex((index) => index + 1)}
        onSubmit={handleSubmit}
      />
    </section>
  );
}

function SurveyQuestionChoices({
  options,
  selected,
  customValue,
  allowFreeText,
  onSelect,
  onCustomChange,
}: {
  options: string[];
  selected: string | null;
  customValue: string;
  allowFreeText: boolean;
  onSelect: (option: string) => void;
  onCustomChange: (event: ChangeEvent<HTMLInputElement>) => void;
}) {
  return (
    <div className="flex flex-col gap-2 px-4 pt-3">
      {options.map((option) => {
        const isSelected = selected === option;
        return (
          <button
            key={option}
            type="button"
            onClick={() => onSelect(option)}
            className={cn(
              "flex w-full items-start gap-3 rounded-md border px-3 py-2.5 text-left text-[13px] leading-snug transition-colors",
              isSelected
                ? "border-foreground/25 bg-muted"
                : "border-border bg-background hover:border-foreground/20 hover:bg-muted/50",
            )}
          >
            <span
              className={cn(
                "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full border",
                isSelected ? "border-foreground bg-foreground" : "border-muted-foreground/40 bg-background",
              )}
              aria-hidden
            >
              {isSelected ? <span className="size-1.5 rounded-full bg-background" /> : null}
            </span>
            {option}
          </button>
        );
      })}
      {allowFreeText ? (
        <Input
          value={customValue}
          onChange={onCustomChange}
          placeholder={WORK_ORDER_SURVEY_CUSTOM_PLACEHOLDER}
          className="h-9 text-[13px]"
        />
      ) : null}
    </div>
  );
}

function SurveyCardNav({
  isFirst,
  isLast,
  canSubmit,
  busy,
  onPrevious,
  onNext,
  onSubmit,
}: {
  isFirst: boolean;
  isLast: boolean;
  canSubmit: boolean;
  busy: boolean;
  onPrevious: () => void;
  onNext: () => void;
  onSubmit: () => void;
}) {
  return (
    <div className="flex items-center justify-between gap-3 px-4 py-4">
      <Button type="button" variant="ghost" size="sm" disabled={isFirst} onClick={onPrevious}>
        <ChevronLeft size={14} aria-hidden />
        Previous
      </Button>
      {isLast ? (
        <Button type="button" size="sm" disabled={!canSubmit || busy} onClick={onSubmit}>
          {WORK_ORDER_SURVEY_SUBMIT}
        </Button>
      ) : (
        <Button type="button" size="sm" onClick={onNext}>
          Next
          <ChevronRight size={14} aria-hidden />
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
