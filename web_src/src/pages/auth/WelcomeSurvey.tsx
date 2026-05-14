import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { posthog, isPostHogEnabled } from "@/posthog";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";

const NEW_USER_ONBOARDING_SURVEY_NAME = "New User Onboarding Survey";

const isValidRedirectPath = (path: string | null): path is string => {
  if (!path || path[0] !== "/") {
    return false;
  }

  if (path.length > 1 && path[1] === "/") {
    return false;
  }

  return path !== "/welcome";
};

const getSafeRedirectPath = (rawRedirect: string | null): string => {
  if (!rawRedirect) {
    return "/";
  }

  try {
    const decoded = decodeURIComponent(rawRedirect);
    return isValidRedirectPath(decoded) ? decoded : "/";
  } catch {
    return "/";
  }
};

const WelcomeSurvey: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [survey, setSurvey] = useState<PostHogSurvey | null>(null);
  const [shouldRedirect, setShouldRedirect] = useState(!isPostHogEnabled);

  const redirectTo = useMemo(() => getSafeRedirectPath(searchParams.get("redirect")), [searchParams]);

  const handleSurveyComplete = useCallback(() => {
    navigate(redirectTo, { replace: true });
  }, [navigate, redirectTo]);

  useEffect(() => {
    if (!isPostHogEnabled) return;

    let canceled = false;

    const selectSurvey = (surveys: PostHogSurvey[]) => {
      const found = surveys.find(
        (survey) =>
          survey.name === NEW_USER_ONBOARDING_SURVEY_NAME &&
          Array.isArray(survey.questions) &&
          survey.questions.length > 0,
      );

      if (!found) {
        setShouldRedirect(true);
        return;
      }

      setSurvey(found);
    };

    const unsubscribe = posthog.onSurveysLoaded(() => {
      posthog.getActiveMatchingSurveys((surveys) => {
        if (canceled) return;
        selectSurvey(surveys as PostHogSurvey[]);
      }, true);
    });

    return () => {
      canceled = true;
      unsubscribe();
    };
  }, []);

  if (shouldRedirect) {
    return <Navigate to={redirectTo} replace />;
  }

  if (!survey) {
    return null;
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4 py-8">
      <div className="max-w-md w-full bg-white dark:bg-gray-900 rounded-lg outline outline-gray-950/10 shadow-sm p-8">
        <PostHogSurveyForm survey={survey} redirectTo={redirectTo} onComplete={handleSurveyComplete} />
      </div>
    </div>
  );
};

export default WelcomeSurvey;
