import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { posthog, isPostHogEnabled } from "@/posthog";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";
import { getWelcomeSurveyRedirectPath } from "./welcomeSurveyRedirect";

const NEW_USER_ONBOARDING_SURVEY_NAME = "New User Onboarding Survey";
const SURVEY_LOAD_FALLBACK_MS = 3000;

const findNewUserSurvey = (surveys: PostHogSurvey[]) =>
  surveys.find(
    (survey) =>
      survey.name === NEW_USER_ONBOARDING_SURVEY_NAME && Array.isArray(survey.questions) && survey.questions.length > 0,
  ) ?? null;

const WelcomeSurvey: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [survey, setSurvey] = useState<PostHogSurvey | null>(null);
  const [shouldRedirect, setShouldRedirect] = useState(!isPostHogEnabled);

  const redirectTo = useMemo(() => getWelcomeSurveyRedirectPath(searchParams.get("redirect")), [searchParams]);

  const handleSurveyComplete = useCallback(() => {
    navigate(redirectTo, { replace: true });
  }, [navigate, redirectTo]);

  useEffect(() => {
    if (!isPostHogEnabled) {
      return;
    }

    let canceled = false;
    let surveyLoadFinished = false;

    const fallbackTimer = window.setTimeout(() => {
      if (!canceled && !surveyLoadFinished) {
        setShouldRedirect(true);
      }
    }, SURVEY_LOAD_FALLBACK_MS);

    const loadMatchingSurveys = () => {
      posthog.getActiveMatchingSurveys((surveys) => {
        if (canceled) {
          return;
        }

        const matchingSurvey = findNewUserSurvey(surveys as PostHogSurvey[]);
        surveyLoadFinished = true;
        window.clearTimeout(fallbackTimer);

        if (!matchingSurvey) {
          setShouldRedirect(true);
          return;
        }

        setSurvey(matchingSurvey);
      }, true);
    };

    const unsubscribe = posthog.onSurveysLoaded(loadMatchingSurveys);
    loadMatchingSurveys();

    return () => {
      canceled = true;
      window.clearTimeout(fallbackTimer);
      if (typeof unsubscribe === "function") {
        unsubscribe();
      }
    };
  }, []);

  if (shouldRedirect) {
    return <Navigate to={redirectTo} replace />;
  }

  if (!survey) {
    return null;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-400 px-4 py-10">
      <div className="w-full max-w-xl rounded-lg bg-white px-8 py-9 shadow-sm outline outline-gray-950/10 dark:bg-gray-900 sm:px-10">
        <PostHogSurveyForm survey={survey} redirectTo={redirectTo} onComplete={handleSurveyComplete} />
      </div>
    </div>
  );
};

export default WelcomeSurvey;
