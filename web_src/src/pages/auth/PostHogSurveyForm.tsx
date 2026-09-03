import React, { useCallback, useEffect, useMemo, useState } from "react";
import { analytics } from "@/lib/analytics";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/ui/checkbox";
import superplaneLogo from "../../assets/superplane.svg";

interface SurveyQuestion {
  id?: string;
  question?: string;
  choices?: string[];
  type?: string;
  display_type?: string;
  allow_multiple?: boolean;
  multiple?: boolean;
  placeholder?: string;
}

export interface PostHogSurvey {
  id: string;
  name: string;
  questions: SurveyQuestion[];
}

interface PostHogSurveyFormProps {
  survey: PostHogSurvey;
  redirectTo: string;
  onComplete?: () => void;
}

type SurveyAnswer = string | string[];
type SurveyResponses = Record<number, SurveyAnswer>;
type SurveyQuestionType = "single_choice" | "multiple_choice" | "text";

const TEXT_QUESTION_PLACEHOLDER = "Describe the task";

const parseChoiceLabel = (choice: string): { title: string; subtitle: string | null } => {
  const match = choice.match(/^(.*?)\s*\(([^)]+)\)\s*$/);
  if (!match) {
    return { title: choice, subtitle: null };
  }

  return { title: match[1].trim(), subtitle: match[2].trim() };
};

const getQuestionType = (question: SurveyQuestion, hasChoices: boolean): SurveyQuestionType => {
  const rawType = `${question.type ?? ""} ${question.display_type ?? ""}`.toLowerCase();
  const isMultiple =
    question.allow_multiple === true || question.multiple === true || /multi|multiple|checkbox/.test(rawType);

  if (isMultiple && hasChoices) {
    return "multiple_choice";
  }

  if (!hasChoices || /open|text|free|long|short/.test(rawType)) {
    return "text";
  }

  return "single_choice";
};

/** Small "A", "B", "C" badge that marks each poll option, matching the AI agent's SurveyWidget. */
const SurveyOptionLetter: React.FC<{ index: number }> = ({ index }) => (
  <span className="mt-0.5 inline-flex size-6 shrink-0 items-center justify-center rounded bg-muted text-xs font-semibold text-muted-foreground">
    {String.fromCharCode(65 + index)}
  </span>
);

const SurveyOptionLabel: React.FC<{ title: string; subtitle: string | null }> = ({ title, subtitle }) => (
  <span className="min-w-0 flex-1 text-sm text-foreground">
    <span className="block font-medium leading-5">{title}</span>
    {subtitle && <span className="mt-1 block font-normal leading-5 text-muted-foreground">{subtitle}</span>}
  </span>
);

interface SurveyChoiceButtonsProps {
  choices: string[];
  onSelect: (choice: string) => void;
}

const SurveyChoiceButtons: React.FC<SurveyChoiceButtonsProps> = ({ choices, onSelect }) => (
  <div className="flex flex-col gap-1.5">
    {choices.map((choice, index) => {
      const { title, subtitle } = parseChoiceLabel(choice);

      return (
        <Button
          key={choice}
          type="button"
          variant="ghost"
          className="h-auto w-full justify-start gap-3 whitespace-normal rounded-md px-3 py-3 text-left hover:bg-muted"
          onClick={() => onSelect(choice)}
        >
          <SurveyOptionLetter index={index} />
          <SurveyOptionLabel title={title} subtitle={subtitle} />
        </Button>
      );
    })}
  </div>
);

interface SurveyMultiChoiceProps {
  choices: string[];
  selectedChoices: string[];
  onToggle: (choice: string, checked: boolean) => void;
  onSubmit: () => void;
}

const SurveyMultiChoice: React.FC<SurveyMultiChoiceProps> = ({ choices, selectedChoices, onToggle, onSubmit }) => (
  <div className="space-y-4">
    <div className="flex flex-col gap-1.5">
      {choices.map((choice, index) => {
        const isChecked = selectedChoices.includes(choice);
        const { title, subtitle } = parseChoiceLabel(choice);

        return (
          <label
            key={choice}
            className={cn(
              "flex cursor-pointer items-start gap-3 rounded-md px-3 py-3 text-left transition-colors",
              isChecked ? "bg-accent ring-1 ring-ring" : "hover:bg-muted",
            )}
          >
            <Checkbox
              checked={isChecked}
              onCheckedChange={(checked) => onToggle(choice, checked === true)}
              className="mt-1 shrink-0"
            />
            <SurveyOptionLetter index={index} />
            {/* Inline (rather than <SurveyOptionLabel />) so the label keeps a direct text
                child; a11y tooling associates the nested Checkbox with this text. */}
            <span className="min-w-0 flex-1 text-sm text-foreground">
              <span className="block font-medium leading-5">{title}</span>
              {subtitle && <span className="mt-1 block font-normal leading-5 text-muted-foreground">{subtitle}</span>}
            </span>
          </label>
        );
      })}
    </div>
    <Button type="button" className="w-full" onClick={onSubmit} disabled={selectedChoices.length === 0}>
      Continue
    </Button>
  </div>
);

interface SurveyTextQuestionProps {
  placeholder?: string;
  textAnswer: string;
  onTextChange: (value: string) => void;
  onSubmit: () => void;
}

const SurveyTextQuestion: React.FC<SurveyTextQuestionProps> = ({ placeholder, textAnswer, onTextChange, onSubmit }) => (
  <div className="space-y-4">
    <Textarea
      placeholder={placeholder ?? TEXT_QUESTION_PLACEHOLDER}
      value={textAnswer}
      onChange={(event) => onTextChange(event.target.value)}
      className="min-h-32 resize-none rounded-md"
    />
    <Button type="button" className="w-full" onClick={onSubmit} disabled={!textAnswer.trim()}>
      Continue
    </Button>
  </div>
);

interface SurveyProgressProps {
  questionCount: number;
  currentQuestionIndex: number;
  onSkip: () => void;
}

const SurveyProgress: React.FC<SurveyProgressProps> = ({ questionCount, currentQuestionIndex, onSkip }) => (
  <div className="flex items-center justify-between gap-4">
    <div className="flex gap-1.5" aria-label={`Question ${currentQuestionIndex + 1} of ${questionCount}`}>
      {Array.from({ length: questionCount }).map((_, i) => (
        <div
          key={i}
          className={cn(
            "h-1 rounded-full transition-all duration-300",
            i === currentQuestionIndex ? "w-8 bg-primary" : "w-4 bg-muted",
          )}
        />
      ))}
    </div>
    <button
      type="button"
      className="text-xs text-muted-foreground transition-colors hover:text-foreground"
      onClick={onSkip}
    >
      Skip
    </button>
  </div>
);

const buildSurveyResponseProps = (survey: PostHogSurvey, responses: SurveyResponses) => {
  const responseProps: Record<string, string | string[]> = {};

  survey.questions.forEach((question, index) => {
    const answer = responses[index];
    if (answer === undefined) {
      return;
    }

    const key = question.id
      ? `$survey_response_${question.id}`
      : index === 0
        ? "$survey_response"
        : `$survey_response_${index}`;

    responseProps[key] = answer;
  });

  return responseProps;
};

const SurveyQuestionHeader: React.FC<{ question: string | undefined }> = ({ question }) => (
  <div className="space-y-5">
    <img src={superplaneLogo} alt="SuperPlane logo" className="h-8 w-8" />
    <div className="space-y-2">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Welcome to SuperPlane</p>
      <h4 className="text-balance text-2xl font-semibold leading-8 text-foreground">{question ?? "Question"}</h4>
    </div>
  </div>
);

const PostHogSurveyForm: React.FC<PostHogSurveyFormProps> = ({ survey, redirectTo, onComplete }) => {
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [surveyResponses, setSurveyResponses] = useState<SurveyResponses>({});
  const [textAnswer, setTextAnswer] = useState("");
  const [multiAnswer, setMultiAnswer] = useState<string[]>([]);

  const currentQuestion = survey.questions[currentQuestionIndex];
  const currentChoices = useMemo(
    () => currentQuestion?.choices?.filter((choice) => Boolean(choice?.trim())) ?? [],
    [currentQuestion],
  );
  const currentType = currentQuestion ? getQuestionType(currentQuestion, currentChoices.length > 0) : "text";

  const handleComplete = useCallback(() => {
    if (onComplete) {
      onComplete();
      return;
    }

    window.location.href = redirectTo;
  }, [onComplete, redirectTo]);

  const finishSurvey = (responses: SurveyResponses) => {
    if (Object.keys(responses).length === 0) {
      analytics.surveyDismissed(survey.id);
    } else {
      analytics.surveySent(survey.id, survey.name, buildSurveyResponseProps(survey, responses));
    }

    handleComplete();
  };

  const advanceOrFinish = (responses: SurveyResponses) => {
    if (currentQuestionIndex < survey.questions.length - 1) {
      setSurveyResponses(responses);
      setCurrentQuestionIndex(currentQuestionIndex + 1);
      setTextAnswer("");
      setMultiAnswer([]);
      return;
    }

    finishSurvey(responses);
  };

  const handleSingleChoice = (answer: string) => {
    const newResponses = { ...surveyResponses, [currentQuestionIndex]: answer };
    advanceOrFinish(newResponses);
  };

  const handleSubmitTextAnswer = () => {
    const answer = textAnswer.trim();
    if (!answer) {
      return;
    }

    const newResponses = { ...surveyResponses, [currentQuestionIndex]: answer };
    advanceOrFinish(newResponses);
  };

  const handleToggleMultiChoice = (choice: string, checked: boolean) => {
    if (checked) {
      setMultiAnswer((previous) => [...previous, choice]);
      return;
    }

    setMultiAnswer((previous) => previous.filter((item) => item !== choice));
  };

  const handleSubmitMultiChoice = () => {
    if (multiAnswer.length === 0) {
      return;
    }

    const newResponses = { ...surveyResponses, [currentQuestionIndex]: multiAnswer };
    advanceOrFinish(newResponses);
  };

  const handleSkipQuestion = () => {
    if (currentQuestionIndex < survey.questions.length - 1) {
      setCurrentQuestionIndex(currentQuestionIndex + 1);
      setTextAnswer("");
      setMultiAnswer([]);
      return;
    }

    finishSurvey(surveyResponses);
  };

  useEffect(() => {
    if (currentQuestion) {
      return;
    }

    analytics.surveyDismissed(survey.id);
    handleComplete();
  }, [currentQuestion, handleComplete, survey.id]);

  if (!currentQuestion) {
    return null;
  }

  return (
    <div className="space-y-7">
      <SurveyQuestionHeader question={currentQuestion.question} />

      {currentType === "single_choice" && (
        <SurveyChoiceButtons choices={currentChoices} onSelect={handleSingleChoice} />
      )}

      {currentType === "multiple_choice" && (
        <SurveyMultiChoice
          choices={currentChoices}
          selectedChoices={multiAnswer}
          onToggle={handleToggleMultiChoice}
          onSubmit={handleSubmitMultiChoice}
        />
      )}

      {currentType === "text" && (
        <SurveyTextQuestion
          placeholder={currentQuestion.placeholder}
          textAnswer={textAnswer}
          onTextChange={setTextAnswer}
          onSubmit={handleSubmitTextAnswer}
        />
      )}

      <SurveyProgress
        questionCount={survey.questions.length}
        currentQuestionIndex={currentQuestionIndex}
        onSkip={handleSkipQuestion}
      />
    </div>
  );
};

export default PostHogSurveyForm;
