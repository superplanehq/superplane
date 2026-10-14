import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { ArrowUp, Check, ChevronLeft, ChevronRight, CircleHelp } from "lucide-react";
import { useEffect, useId, useRef, useState, type KeyboardEvent } from "react";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentSurvey } from "./createWithAgentTypes";
import { formatPlanningSurveyReply } from "./planningSessionSurvey";

const SKIPPED_ANSWER = Symbol("skipped answer");
type SurveyAnswer = string | typeof SKIPPED_ANSWER | null;

export function PlanningSessionSurveyForm({
  survey,
  onSubmit,
}: {
  survey: CreateWithAgentSurvey;
  onSubmit: (text: string) => void;
}) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<SurveyAnswer[]>(() => survey.questions.map(() => null));
  const [customInputs, setCustomInputs] = useState<string[]>(() => survey.questions.map(() => ""));
  const questionRef = useRef<HTMLParagraphElement>(null);
  const previousIndex = useRef(currentIndex);
  useEffect(() => {
    if (previousIndex.current !== currentIndex) questionRef.current?.focus();
    previousIndex.current = currentIndex;
  }, [currentIndex]);
  const question = survey.questions[currentIndex];
  const questionCount = survey.questions.length;
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === questionCount - 1;
  const hasCurrentAnswer = Boolean(answers[currentIndex]);
  const submitAnswers = (selectedAnswers: SurveyAnswer[]) =>
    onSubmit(
      formatPlanningSurveyReply(
        survey.questions,
        selectedAnswers.map((answer) => (answer === SKIPPED_ANSWER ? null : answer)),
      ),
    );
  const sendReply = () => submitAnswers(answers);
  const selectAnswer = (answer: Exclude<SurveyAnswer, null>) => {
    const alreadySelected = answers[currentIndex] === answer;
    if (questionCount === 1 && alreadySelected) {
      return;
    }
    const nextAnswers = replaceAtIndex(answers, currentIndex, alreadySelected ? null : answer);
    setAnswers(nextAnswers);
    setCustomInputs((current) => replaceAtIndex(current, currentIndex, ""));
    if (questionCount === 1) {
      submitAnswers(nextAnswers);
      return;
    }
    if (!alreadySelected && !isLast) {
      setCurrentIndex((index) => index + 1);
    }
  };

  if (!question) {
    return null;
  }

  return (
    <div className="sp-survey-enter mt-3 mb-1" data-testid="create-with-agent-survey">
      <div className="rounded-xl border bg-card p-3.5 shadow-xs" data-testid="create-with-agent-survey-card">
        {questionCount > 1 ? (
          <p className="text-xs font-medium text-muted-foreground">
            Question {currentIndex + 1} of {questionCount}
          </p>
        ) : null}
        <div key={question.prompt} className="sp-survey-page">
          <p
            ref={questionRef}
            tabIndex={-1}
            className={cn(
              "flex items-start gap-2 text-sm font-semibold text-foreground",
              questionCount > 1 && "mt-1.5",
            )}
          >
            <CircleHelp className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
            {question.prompt}
          </p>
          <div className="mt-3 divide-y divide-border/70 overflow-hidden rounded-lg border border-border">
            {question.options.map((option, index) => (
              <SurveyOptionButton
                key={option}
                badge={String.fromCharCode(65 + index)}
                option={option}
                selected={answers[currentIndex] === option}
                onSelect={() => selectAnswer(option)}
              />
            ))}
            <SurveyOptionButton
              badge={String.fromCharCode(65 + question.options.length)}
              option={CREATE_WITH_AGENT_COPY.skipSurvey}
              selected={answers[currentIndex] === SKIPPED_ANSWER}
              onSelect={() => selectAnswer(SKIPPED_ANSWER)}
            />
          </div>
          <SurveyAnswerComposer
            prompt={question.prompt}
            value={customInputs[currentIndex] ?? ""}
            hasCurrentAnswer={hasCurrentAnswer}
            showSend={questionCount === 1}
            onChange={(value) => {
              setCustomInputs((current) => replaceAtIndex(current, currentIndex, value));
              setAnswers((current) => replaceAtIndex(current, currentIndex, value.trim() || null));
            }}
            onKeyDown={(event) =>
              handleSurveyEnter(event, isLast, hasCurrentAnswer, () => setCurrentIndex((index) => index + 1), sendReply)
            }
            onSend={sendReply}
          />
        </div>
        {questionCount > 1 ? (
          <SurveyFormPager
            questionCount={questionCount}
            isFirst={isFirst}
            isLast={isLast}
            canSend={hasCurrentAnswer}
            onPrevious={() => setCurrentIndex((index) => index - 1)}
            onNext={() => setCurrentIndex((index) => index + 1)}
            onSend={sendReply}
          />
        ) : null}
      </div>
    </div>
  );
}

function SurveyAnswerComposer({
  prompt,
  value,
  hasCurrentAnswer,
  showSend,
  onChange,
  onKeyDown,
  onSend,
}: {
  prompt: string;
  value: string;
  hasCurrentAnswer: boolean;
  showSend: boolean;
  onChange: (value: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
  onSend: () => void;
}) {
  const inputId = useId();

  return (
    <div className="mt-3">
      <Label htmlFor={inputId} className="sr-only">
        {CREATE_WITH_AGENT_COPY.otherAnswer}
      </Label>
      <InputGroup className="h-10 rounded-lg border-border bg-background shadow-none dark:bg-background">
        <InputGroupInput
          id={inputId}
          name="planning-survey-answer"
          type="text"
          value={value}
          placeholder={CREATE_WITH_AGENT_COPY.otherAnswer}
          aria-label={`${prompt} ${CREATE_WITH_AGENT_COPY.otherAnswer}`}
          className="h-10 min-w-0 px-3 text-sm text-foreground placeholder:text-muted-foreground"
          onChange={(event) => onChange(event.target.value)}
          onKeyDown={onKeyDown}
        />
        {showSend ? (
          <InputGroupAddon align="inline-end" className="pr-1">
            <InputGroupButton
              type="button"
              variant="secondary"
              size="icon-sm"
              className="rounded-md"
              aria-label={CREATE_WITH_AGENT_COPY.sendAnswer}
              disabled={!hasCurrentAnswer}
              onClick={onSend}
            >
              <ArrowUp className="size-4" aria-hidden />
            </InputGroupButton>
          </InputGroupAddon>
        ) : null}
      </InputGroup>
    </div>
  );
}

function SurveyOptionButton({
  badge,
  option,
  selected,
  onSelect,
}: {
  badge: string;
  option: string;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-pressed={selected}
      className={cn(
        "h-auto min-h-10 w-full justify-between gap-2 whitespace-normal rounded-none px-3 py-2 text-left text-sm font-normal shadow-none",
        selected ? "bg-muted/70 text-foreground hover:bg-muted/70" : "bg-background text-foreground hover:bg-muted/50",
      )}
      onClick={onSelect}
    >
      <span className="flex min-w-0 items-center gap-2.5">
        <SurveyChoiceBadge label={badge} />
        <span>{option}</span>
      </span>
      {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
    </Button>
  );
}

function SurveyChoiceBadge({ label }: { label: string }) {
  return (
    <span
      aria-hidden
      className="flex size-6 shrink-0 items-center justify-center rounded-full bg-foreground/10 text-[13px] font-bold text-foreground"
    >
      {label}
    </span>
  );
}

function handleSurveyEnter(
  event: KeyboardEvent<HTMLInputElement>,
  isLast: boolean,
  canSend: boolean,
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
  if (canSend) {
    onSend();
  }
}

const quietActionClassName = "h-8 text-sm text-foreground/75";

function SurveyFormPager({
  questionCount,
  isFirst,
  isLast,
  canSend,
  onPrevious,
  onNext,
  onSend,
}: {
  questionCount: number;
  isFirst: boolean;
  isLast: boolean;
  canSend: boolean;
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
          className={quietActionClassName}
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
        <LastPageActionButton questionCount={questionCount} canSend={canSend} onSend={onSend} />
      ) : (
        <Button type="button" variant="ghost" size="sm" className={quietActionClassName} onClick={onNext}>
          {CREATE_WITH_AGENT_COPY.nextQuestion}
          <ChevronRight size={12} className="ml-1" />
        </Button>
      )}
    </div>
  );
}

function LastPageActionButton({
  questionCount,
  canSend,
  onSend,
}: {
  questionCount: number;
  canSend: boolean;
  onSend: () => void;
}) {
  return (
    <Button
      type="button"
      variant="default"
      size="sm"
      className="h-8 shrink-0 text-sm"
      disabled={!canSend}
      onClick={onSend}
    >
      {questionCount === 1 ? CREATE_WITH_AGENT_COPY.sendAnswer : CREATE_WITH_AGENT_COPY.sendAnswers}
    </Button>
  );
}

function replaceAtIndex<T>(items: T[], index: number, value: T): T[] {
  const next = [...items];
  next[index] = value;
  return next;
}
