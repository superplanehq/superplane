import { useState } from "react";
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

  if (!questions.length || submitted) return null;

  const current = questions[currentIndex];
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === questions.length - 1;

  function updateAnswer(value: string | null) {
    setAnswers((current) => replaceAtIndex(current, currentIndex, value));
  }

  function updateCustomInput(value: string) {
    setCustomInputs((current) => replaceAtIndex(current, currentIndex, value));
    if (value.trim()) {
      updateAnswer(value.trim());
    }
  }

  function selectOption(option: string) {
    updateAnswer(option);
  }

  function handleSubmit() {
    if (submitted) return;
    setSubmitted(true);
    const parts = questions.map((question, index) => `${question.prompt} → ${answers[index] || "skipped"}`);
    onAction?.(parts.join("\n"));
  }

  return (
    <div className="my-4 rounded-lg border border-violet-200 bg-white shadow-sm overflow-hidden">
      {/* Header */}
      <div className="px-3 py-2 bg-violet-50 border-b border-violet-200 flex items-center justify-between">
        <p className="text-xs font-medium text-violet-900">{current.prompt}</p>
        <span className="text-[10px] text-violet-500 font-medium">
          {currentIndex + 1}/{questions.length}
        </span>
      </div>

      {/* Dot indicators */}
      <div className="px-3 pt-2 flex items-center gap-1">
        {questions.map((_, i) => (
          <button
            key={i}
            type="button"
            onClick={() => setCurrentIndex(i)}
            className={cn(
              "size-2 rounded-full transition-colors",
              i === currentIndex ? "bg-violet-600" : answers[i] !== null ? "bg-violet-300" : "bg-slate-200",
            )}
            aria-label={`Question ${i + 1}`}
          />
        ))}
      </div>

      {/* Options */}
      <div className="p-2 flex flex-col gap-1.5">
        {current.options.map((option, i) => (
          <Button
            key={option}
            variant="ghost"
            size="sm"
            className={cn(
              "justify-start text-xs h-auto py-2 px-3 text-left whitespace-normal",
              answers[currentIndex] === option
                ? "bg-violet-100 text-violet-900 ring-1 ring-violet-300"
                : "text-slate-700 hover:bg-violet-50 hover:text-violet-900",
            )}
            onClick={() => selectOption(option)}
          >
            <span className="inline-flex items-center justify-center size-5 rounded bg-violet-100 text-violet-700 text-[10px] font-semibold mr-2 shrink-0">
              {String.fromCharCode(65 + i)}
            </span>
            {option}
          </Button>
        ))}

        {/* Custom input option */}
        {current.hasInput && (
          <div className="flex items-center gap-2 mt-1">
            <input
              type="text"
              placeholder="Type your own answer..."
              value={customInputs[currentIndex]}
              onChange={(e) => updateCustomInput(e.target.value)}
              className={cn(
                "flex-1 text-xs px-3 py-2 rounded border transition-colors outline-none",
                customInputs[currentIndex] && answers[currentIndex] === customInputs[currentIndex].trim()
                  ? "border-violet-300 bg-violet-50 ring-1 ring-violet-300"
                  : "border-slate-200 bg-white focus:border-violet-300",
              )}
            />
          </div>
        )}
      </div>

      {/* Navigation + Submit */}
      <div className="px-3 pb-3 pt-1 flex items-center justify-between">
        <Button
          variant="ghost"
          size="sm"
          className="text-xs text-slate-500 h-7"
          disabled={isFirst}
          onClick={() => setCurrentIndex((i) => i - 1)}
        >
          <ChevronLeft size={12} className="mr-1" />
          Prev
        </Button>

        {isLast ? (
          <Button size="sm" className="text-xs h-7 bg-violet-600 hover:bg-violet-700 text-white" onClick={handleSubmit}>
            Continue →
          </Button>
        ) : (
          <Button
            variant="ghost"
            size="sm"
            className="text-xs text-slate-500 h-7"
            onClick={() => setCurrentIndex((i) => i + 1)}
          >
            Next
            <ChevronRight size={12} className="ml-1" />
          </Button>
        )}
      </div>
    </div>
  );
}

function replaceAtIndex<T>(items: T[], index: number, value: T): T[] {
  const next = [...items];
  next[index] = value;
  return next;
}
