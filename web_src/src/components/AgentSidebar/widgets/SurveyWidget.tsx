import { useCallback, useState, type ChangeEvent } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ChevronLeft, ChevronRight } from "lucide-react";

export interface SurveyQuestion {
  prompt: string;
  options: string[];
  hasInput?: boolean;
}

interface SurveyWidgetProps {
  questions: SurveyQuestion[];
  onAction?: (text: string) => void;
}

export function SurveyWidget({ questions, onAction }: SurveyWidgetProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<(string | null)[]>(() => questions.map(() => null));
  const [customInputs, setCustomInputs] = useState<string[]>(() => questions.map(() => ""));
  const [submitted, setSubmitted] = useState(false);

  const updateAnswer = useCallback(
    (value: string | null) => {
      setAnswers((current) => replaceAtIndex(current, currentIndex, value));
    },
    [currentIndex],
  );

  const updateCustomInput = useCallback(
    (value: string) => {
      setCustomInputs((current) => replaceAtIndex(current, currentIndex, value));
      if (value.trim()) {
        updateAnswer(value.trim());
      }
    },
    [currentIndex, updateAnswer],
  );

  const selectOption = useCallback(
    (option: string) => {
      updateAnswer(option);
    },
    [updateAnswer],
  );

  const handleQuestionSelect = useCallback((index: number) => {
    setCurrentIndex(index);
  }, []);

  const handlePrevious = useCallback(() => {
    setCurrentIndex((index) => index - 1);
  }, []);

  const handleNext = useCallback(() => {
    setCurrentIndex((index) => index + 1);
  }, []);

  const handleCustomInputChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      updateCustomInput(event.target.value);
    },
    [updateCustomInput],
  );

  const handleSubmit = useCallback(() => {
    if (submitted) return;
    setSubmitted(true);
    const parts = questions.map((question, index) => `${question.prompt} → ${answers[index] || "skipped"}`);
    onAction?.(parts.join("\n"));
  }, [answers, onAction, questions, submitted]);

  if (!questions.length || submitted) return null;

  const current = questions[currentIndex];
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === questions.length - 1;

  return (
    <div className="my-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900/60">
        <p className="text-xs font-medium text-slate-900 dark:text-gray-100">{current.prompt}</p>
        <span className="text-[10px] font-medium text-slate-500 dark:text-gray-400">
          {currentIndex + 1}/{questions.length}
        </span>
      </div>

      {/* Dot indicators */}
      <SurveyStepDots
        questionCount={questions.length}
        currentIndex={currentIndex}
        answers={answers}
        onSelect={handleQuestionSelect}
      />

      {/* Options */}
      <div className="flex flex-col gap-1.5 p-2">
        {current.options.map((option, i) => (
          <SurveyOptionButton
            key={option}
            index={i}
            option={option}
            selected={answers[currentIndex] === option}
            onSelect={selectOption}
          />
        ))}

        {/* Custom input option */}
        {current.hasInput && (
          <div className="mt-1 flex items-center gap-2">
            <input
              type="text"
              placeholder="Type your own answer..."
              value={customInputs[currentIndex]}
              onChange={handleCustomInputChange}
              className={cn(
                "flex-1 rounded border px-3 py-2 text-xs outline-none transition-colors dark:text-gray-100 dark:placeholder:text-gray-500",
                customInputs[currentIndex] && answers[currentIndex] === customInputs[currentIndex].trim()
                  ? "border-slate-400 bg-slate-50 ring-1 ring-slate-300 dark:border-gray-500 dark:bg-gray-700 dark:ring-gray-600"
                  : "border-slate-200 bg-white focus:border-slate-400 dark:border-gray-700 dark:bg-gray-900 dark:focus:border-gray-500",
              )}
            />
          </div>
        )}
      </div>

      {/* Navigation + Submit */}
      <SurveyNavigation
        isFirst={isFirst}
        isLast={isLast}
        onPrevious={handlePrevious}
        onNext={handleNext}
        onSubmit={handleSubmit}
      />
    </div>
  );
}

function SurveyStepDots({
  questionCount,
  currentIndex,
  answers,
  onSelect,
}: {
  questionCount: number;
  currentIndex: number;
  answers: Array<string | null>;
  onSelect: (index: number) => void;
}) {
  return (
    <div className="flex items-center gap-1 px-3 pt-2">
      {Array.from({ length: questionCount }, (_, index) => (
        <SurveyStepDot
          key={index}
          index={index}
          isActive={index === currentIndex}
          isAnswered={answers[index] !== null}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}

function SurveyStepDot({
  index,
  isActive,
  isAnswered,
  onSelect,
}: {
  index: number;
  isActive: boolean;
  isAnswered: boolean;
  onSelect: (index: number) => void;
}) {
  const handleClick = useCallback(() => {
    onSelect(index);
  }, [index, onSelect]);

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "size-2 rounded-full transition-colors",
        isActive
          ? "bg-slate-700 dark:bg-gray-300"
          : isAnswered
            ? "bg-slate-400 dark:bg-gray-500"
            : "bg-slate-200 dark:bg-gray-700",
      )}
      aria-label={`Question ${index + 1}`}
    />
  );
}

function SurveyOptionButton({
  index,
  option,
  selected,
  onSelect,
}: {
  index: number;
  option: string;
  selected: boolean;
  onSelect: (option: string) => void;
}) {
  const handleClick = useCallback(() => {
    onSelect(option);
  }, [onSelect, option]);

  return (
    <Button
      variant="ghost"
      size="sm"
      className={cn(
        "h-auto justify-start whitespace-normal px-3 py-2 text-left text-xs",
        selected
          ? "bg-slate-100 text-slate-900 ring-1 ring-slate-300 dark:bg-gray-700 dark:text-gray-100 dark:ring-gray-600"
          : "text-slate-700 hover:bg-slate-50 hover:text-slate-900 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-gray-100",
      )}
      onClick={handleClick}
    >
      <span className="mr-2 inline-flex size-5 shrink-0 items-center justify-center rounded bg-slate-100 text-[10px] font-semibold text-slate-700 dark:bg-gray-700 dark:text-gray-200">
        {String.fromCharCode(65 + index)}
      </span>
      {option}
    </Button>
  );
}

function SurveyNavigation({
  isFirst,
  isLast,
  onPrevious,
  onNext,
  onSubmit,
}: {
  isFirst: boolean;
  isLast: boolean;
  onPrevious: () => void;
  onNext: () => void;
  onSubmit: () => void;
}) {
  return (
    <div className="flex items-center justify-between px-3 pb-3 pt-1">
      <Button
        variant="ghost"
        size="sm"
        className="h-7 text-xs text-slate-500 dark:text-gray-400"
        disabled={isFirst}
        onClick={onPrevious}
      >
        <ChevronLeft size={12} className="mr-1" />
        Prev
      </Button>

      {isLast ? (
        <Button size="sm" className="h-7 text-xs" onClick={onSubmit}>
          Continue →
        </Button>
      ) : (
        <Button variant="ghost" size="sm" className="h-7 text-xs text-slate-500 dark:text-gray-400" onClick={onNext}>
          Next
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
