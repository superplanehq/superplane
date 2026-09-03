import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";

const surveyCapture = vi.hoisted(() => vi.fn());

vi.mock("@/posthog", () => ({
  posthog: { capture: surveyCapture },
}));

const singleChoiceQuestion = {
  id: "q1",
  question: "What best describes your role?",
  type: "single_choice",
  choices: ["Software engineer (writes and ships code)", "Product manager"],
};

const multipleChoiceQuestion = {
  id: "q2",
  question: "Which tools does your team use today?",
  type: "multiple_choice",
  allow_multiple: true,
  choices: ["GitHub", "GitHub Actions"],
};

const textQuestion = {
  id: "q3",
  question: "If you had a single task for an AI agent on your software development process today, what would it be?",
  type: "open",
};

function buildSurvey(questions: PostHogSurvey["questions"]): PostHogSurvey {
  return { id: "survey-1", name: "New User Onboarding Survey", questions };
}

describe("PostHogSurveyForm", () => {
  it("renders single choice options as a poll and advances on selection", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();

    render(
      <PostHogSurveyForm
        survey={buildSurvey([singleChoiceQuestion, textQuestion])}
        redirectTo="/"
        onComplete={onComplete}
      />,
    );

    expect(screen.getByText("What best describes your role?")).toBeInTheDocument();
    expect(screen.getByText("A")).toBeInTheDocument();
    expect(screen.getByText("writes and ships code")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /software engineer/i }));

    expect(
      screen.getByText(
        "If you had a single task for an AI agent on your software development process today, what would it be?",
      ),
    ).toBeInTheDocument();
  });

  it("requires at least one selection before continuing on multiple choice questions", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();

    render(<PostHogSurveyForm survey={buildSurvey([multipleChoiceQuestion])} redirectTo="/" onComplete={onComplete} />);

    const continueButton = screen.getByRole("button", { name: "Continue" });
    expect(continueButton).toBeDisabled();

    await user.click(screen.getByRole("checkbox", { name: /github actions/i }));
    expect(continueButton).toBeEnabled();

    await user.click(continueButton);

    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(surveyCapture).toHaveBeenCalledWith(
      "survey sent",
      expect.objectContaining({
        $survey_response_q2: ["GitHub Actions"],
        $survey_completed: true,
      }),
    );
  });

  it("submits the final open text answer and reports the survey as sent", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();

    render(<PostHogSurveyForm survey={buildSurvey([textQuestion])} redirectTo="/" onComplete={onComplete} />);

    const textarea = screen.getByPlaceholderText("Describe the task");
    await user.type(textarea, "Review my pull request before I merge it.");
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(surveyCapture).toHaveBeenCalledWith(
      "survey sent",
      expect.objectContaining({
        $survey_response_q3: "Review my pull request before I merge it.",
      }),
    );
  });

  it("skips the current question and reports the survey as dismissed when no answers were given", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();

    render(<PostHogSurveyForm survey={buildSurvey([singleChoiceQuestion])} redirectTo="/" onComplete={onComplete} />);

    await user.click(screen.getByRole("button", { name: "Skip" }));

    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(surveyCapture).toHaveBeenCalledWith("survey dismissed", { $survey_id: "survey-1" });
  });
});
