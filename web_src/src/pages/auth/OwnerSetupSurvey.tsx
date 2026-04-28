import React, { useState } from "react";
import { analytics } from "@/lib/analytics";
import superplaneLogo from "../../assets/superplane.svg";
import { Button } from "@/components/ui/button";
import {
  Github,
  Video,
  MessageSquare,
  GraduationCap,
  Trophy,
  Plug,
  Users,
  Sparkles,
  type LucideIcon,
} from "lucide-react";

interface SurveyQuestion {
  id?: string;
  question: string;
  choices?: string[];
}

export interface PostHogSurvey {
  id: string;
  name: string;
  questions: SurveyQuestion[];
}

interface OwnerSetupSurveyProps {
  survey: PostHogSurvey;
  organizationId: string;
}

// Parse "Title (subtitle text)" into { title, subtitle }
const parseChoice = (choice: string): { title: string; subtitle: string | null } => {
  const match = choice.match(/^(.*?)\s*\(([^)]+)\)\s*$/);
  if (match) {
    return { title: match[1].trim(), subtitle: match[2].trim() };
  }
  return { title: choice, subtitle: null };
};

const ICON_MAP: Array<{ keywords: string[]; icon: LucideIcon }> = [
  { keywords: ["github"], icon: Github },
  { keywords: ["dev", "content", "video", "blog", "tutorial"], icon: Video },
  { keywords: ["social", "media", "linkedin", "discord"], icon: MessageSquare },
  { keywords: ["university", "campus", "student"], icon: GraduationCap },
  { keywords: ["hackathon", "community", "meetup", "event"], icon: Trophy },
  { keywords: ["partner", "integration"], icon: Plug },
  { keywords: ["referral", "friend", "colleague", "recommend"], icon: Users },
];

// Match icon based on the title only (not the subtitle) to avoid false positives
const getChoiceIcon = (title: string): LucideIcon => {
  const lower = title.toLowerCase();
  return ICON_MAP.find(({ keywords }) => keywords.some((k) => lower.includes(k)))?.icon ?? Sparkles;
};

interface SurveyIntroProps {
  onStart: () => void;
  onSkip: () => void;
}

const SurveyIntro: React.FC<SurveyIntroProps> = ({ onStart, onSkip }) => (
  <div className="space-y-6">
    <div className="text-center">
      <img src={superplaneLogo} alt="SuperPlane logo" className="mx-auto mb-4 h-8 w-8" />
      <h4 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">Help us get to know you</h4>
      <p className="text-sm text-gray-500 dark:text-gray-400">
        A couple of quick questions to help us understand our community better.
      </p>
    </div>
    <div className="space-y-3">
      <Button className="w-full" onClick={onStart}>
        Let&apos;s go
      </Button>
      <Button variant="outline" className="w-full" onClick={onSkip}>
        Skip for now
      </Button>
    </div>
    <p className="text-center text-xs text-gray-400 dark:text-gray-500">
      Takes less than a minute. You can skip any question.
    </p>
  </div>
);

interface SurveyQuestionStepProps {
  question: SurveyQuestion;
  questionIndex: number;
  totalQuestions: number;
  showIcons: boolean;
  onAnswer: (answer: string) => void;
  onSkip: () => void;
}

const SurveyQuestionStep: React.FC<SurveyQuestionStepProps> = ({
  question,
  questionIndex,
  totalQuestions,
  showIcons,
  onAnswer,
  onSkip,
}) => (
  <div className="space-y-6">
    <div className="text-center">
      <img src={superplaneLogo} alt="SuperPlane logo" className="mx-auto mb-4 h-8 w-8" />
      <h4 className="text-lg font-semibold text-gray-900 dark:text-white">{question.question}</h4>
    </div>
    <div className="space-y-2">
      {question.choices?.map((choice) => {
        const { title, subtitle } = parseChoice(choice);
        const Icon = getChoiceIcon(title);
        return (
          <button
            key={choice}
            type="button"
            onClick={() => onAnswer(choice)}
            className="w-full flex items-center gap-3 px-4 py-3 rounded-lg border border-gray-200 hover:border-gray-900 hover:bg-gray-50 dark:border-gray-700 dark:hover:border-gray-200 dark:hover:bg-gray-800 text-left transition-colors group"
          >
            {showIcons && (
              <div className="flex-shrink-0 w-9 h-9 rounded-md bg-gray-100 dark:bg-gray-800 group-hover:bg-gray-200 dark:group-hover:bg-gray-700 flex items-center justify-center transition-colors">
                <Icon className="w-4 h-4 text-gray-600 dark:text-gray-400" />
              </div>
            )}
            <div className="flex flex-col">
              <span className="text-sm font-medium text-gray-800 dark:text-gray-200">{title}</span>
              {subtitle && <span className="text-xs text-gray-500 dark:text-gray-400">{subtitle}</span>}
            </div>
          </button>
        );
      })}
    </div>
    <div className="flex flex-col items-center gap-3">
      <div className="flex gap-1.5">
        {Array.from({ length: totalQuestions }).map((_, i) => (
          <div
            key={i}
            className={`h-1.5 rounded-full transition-all duration-300 ${
              i === questionIndex ? "w-4 bg-gray-900 dark:bg-white" : "w-1.5 bg-gray-300 dark:bg-gray-600"
            }`}
          />
        ))}
      </div>
      <button
        type="button"
        className="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
        onClick={onSkip}
      >
        Skip this question
      </button>
    </div>
  </div>
);

const OwnerSetupSurvey: React.FC<OwnerSetupSurveyProps> = ({ survey, organizationId }) => {
  const [surveyStep, setSurveyStep] = useState<"intro" | "questions">("intro");
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [surveyResponses, setSurveyResponses] = useState<Record<number, string>>({});

  const handleSkipAll = () => {
    analytics.surveyDismissed(survey.id);
    window.location.href = `/${organizationId}`;
  };

  const finishSurvey = (responses: Record<number, string>) => {
    if (Object.keys(responses).length === 0) {
      analytics.surveyDismissed(survey.id);
    } else {
      const responseProps: Record<string, string> = {};
      survey.questions.forEach((question, index) => {
        const answer = responses[index];
        if (answer === undefined) return;
        if (question.id) {
          responseProps[`$survey_response_${question.id}`] = answer;
        } else if (index === 0) {
          responseProps["$survey_response"] = answer;
        } else {
          responseProps[`$survey_response_${index}`] = answer;
        }
      });
      analytics.surveySent(survey.id, survey.name, responseProps);
    }
    window.location.href = `/${organizationId}`;
  };

  const handleAnswerQuestion = (answer: string) => {
    const newResponses = { ...surveyResponses, [currentQuestionIndex]: answer };
    if (currentQuestionIndex < survey.questions.length - 1) {
      setSurveyResponses(newResponses);
      setCurrentQuestionIndex(currentQuestionIndex + 1);
    } else {
      finishSurvey(newResponses);
    }
  };

  const handleSkipQuestion = () => {
    if (currentQuestionIndex < survey.questions.length - 1) {
      setCurrentQuestionIndex(currentQuestionIndex + 1);
    } else {
      finishSurvey(surveyResponses);
    }
  };

  if (surveyStep === "intro") {
    return <SurveyIntro onStart={() => setSurveyStep("questions")} onSkip={handleSkipAll} />;
  }

  return (
    <SurveyQuestionStep
      question={survey.questions[currentQuestionIndex]}
      questionIndex={currentQuestionIndex}
      totalQuestions={survey.questions.length}
      showIcons={currentQuestionIndex === 0}
      onAnswer={handleAnswerQuestion}
      onSkip={handleSkipQuestion}
    />
  );
};

export default OwnerSetupSurvey;
