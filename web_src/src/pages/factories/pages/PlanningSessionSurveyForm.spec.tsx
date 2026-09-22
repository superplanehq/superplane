import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import { PlanningSessionSurveyForm } from "./PlanningSessionSurveyForm";

const twoQuestions = {
  questions: [
    { prompt: "What is the priority?", options: ["High", "Low"] },
    { prompt: "What is the scope?", options: ["One file", "The service"] },
  ],
};

describe("PlanningSessionSurveyForm", () => {
  it("shows one question at a time and pages with Next and Previous", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    expect(screen.getByText(CREATE_WITH_AGENT_COPY.surveyHeader)).toBeInTheDocument();
    expect(screen.getByTestId("create-with-agent-survey-card")).toHaveClass("border", "bg-card");
    expect(screen.getByText("What is the priority?")).toBeInTheDocument();
    expect(screen.queryByText("What is the scope?")).not.toBeInTheDocument();
    expect(screen.getByText("1 of 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /High/ })).toHaveTextContent("A");

    const high = screen.getByRole("button", { name: /High/ });
    await user.click(high);
    expect(high).toHaveAttribute("aria-pressed", "true");
    expect(high).toHaveClass("bg-primary", "text-primary-foreground");
    expect(high).not.toHaveClass("dark:text-gray-300");
    expect(screen.getByRole("button", { name: /Low/ })).toHaveClass("bg-background");
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
    expect(screen.queryByText("What is the priority?")).not.toBeInTheDocument();
    expect(screen.getByText("2 of 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers })).toBeDisabled();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion }));

    expect(screen.getByText("What is the priority?")).toBeInTheDocument();
  });

  it("does not send when Next is followed by the primary action before the next question is answered", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers }));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
  });

  it("clears the current choice on a second click", async () => {
    const user = userEvent.setup();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={vi.fn()} />);

    const high = screen.getByRole("button", { name: /High/ });
    await user.click(high);
    await user.click(high);

    expect(high).toHaveAttribute("aria-pressed", "false");
    expect(high).toHaveClass("bg-background", "text-foreground");
    expect(high).not.toHaveClass("bg-primary");
  });

  it("replaces the current choice when a different option is clicked", async () => {
    const user = userEvent.setup();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={vi.fn()} />);

    const high = screen.getByRole("button", { name: /High/ });
    const low = screen.getByRole("button", { name: /Low/ });
    await user.click(high);
    await user.click(low);

    expect(high).toHaveAttribute("aria-pressed", "false");
    expect(low).toHaveAttribute("aria-pressed", "true");
    expect(low).toHaveClass("bg-primary", "text-primary-foreground");
    expect(low).not.toHaveClass("dark:text-gray-300");
  });

  it("keeps send disabled after the last choice is cleared", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));
    const oneFile = screen.getByRole("button", { name: /One file/ });
    await user.click(oneFile);
    await user.click(oneFile);

    expect(oneFile).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers })).toBeDisabled();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("hides page controls when there is one question", () => {
    render(
      <PlanningSessionSurveyForm
        survey={{ questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }] }}
        onSubmit={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toBeInTheDocument();
  });

  it("uses Enter on a middle question as Next, not Submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.type(screen.getByRole("textbox"), "Custom{Enter}");

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
  });

  it("sends answered and skipped questions from Skip on the last page", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey }));

    expect(onSubmit).toHaveBeenCalledWith("What is the priority? High\nWhat is the scope? skipped");
  });

  it("sends every answer after the last question is picked", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));
    await user.click(screen.getByRole("button", { name: /One file/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers }));

    expect(onSubmit).toHaveBeenCalledWith("What is the priority? High\nWhat is the scope? One file");
  });

  it("resets pages when a new survey mounts", () => {
    const { rerender } = render(<PlanningSessionSurveyForm key="old" survey={twoQuestions} onSubmit={vi.fn()} />);
    rerender(
      <PlanningSessionSurveyForm
        key="new"
        survey={{ questions: [{ prompt: "Only one?", options: ["Yes"] }] }}
        onSubmit={vi.fn()}
      />,
    );

    expect(screen.getByText("Only one?")).toBeInTheDocument();
    expect(screen.queryByText("What is the priority?")).not.toBeInTheDocument();
  });
});
