import { render, screen, within } from "@testing-library/react";
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
  it("advances after a choice and keeps that choice when navigating back", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    expect(screen.getByText("What is the priority?")).toBeInTheDocument();
    expect(screen.queryByText("What is the scope?")).not.toBeInTheDocument();
    expect(screen.getByText("Question 1 of 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toBeInTheDocument();

    const high = screen.getByRole("button", { name: /High/ });
    await user.click(high);

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
    expect(screen.queryByText("What is the priority?")).not.toBeInTheDocument();
    expect(screen.getByText("Question 2 of 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers })).toBeDisabled();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion }));

    expect(screen.getByText("What is the priority?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /High/ })).toHaveAttribute("aria-pressed", "true");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("moves keyboard focus to the next question after a choice", async () => {
    const user = userEvent.setup();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={vi.fn()} />);

    await user.tab();
    expect(screen.getByRole("button", { name: "High" })).toHaveFocus();

    await user.keyboard("{Enter}");

    expect(screen.getByText("What is the scope?")).toHaveFocus();
    await user.tab();
    expect(screen.getByRole("button", { name: "One file" })).toHaveFocus();
  });

  it("keeps Next as navigation without submitting an unanswered question", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

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
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion }));
    await user.click(screen.getByRole("button", { name: /High/ }));

    expect(screen.getByRole("button", { name: /High/ })).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByText("Question 1 of 2")).toBeInTheDocument();
  });

  it("replaces the current choice when a different option is clicked", async () => {
    const user = userEvent.setup();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion }));
    await user.click(screen.getByRole("button", { name: /Low/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion }));

    expect(screen.getByRole("button", { name: /High/ })).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("button", { name: /Low/ })).toHaveAttribute("aria-pressed", "true");
  });

  it("clears a typed answer when an option is selected", async () => {
    const user = userEvent.setup();
    render(<PlanningSessionSurveyForm survey={{ questions: [twoQuestions.questions[0]] }} onSubmit={vi.fn()} />);

    const answer = screen.getByRole("textbox", { name: /What is the priority/ });
    await user.type(answer, "Ask the agent to decide");
    await user.click(screen.getByRole("button", { name: /High/ }));

    expect(answer).toHaveValue("");
    expect(screen.getByRole("button", { name: /High/ })).toHaveAttribute("aria-pressed", "true");
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

  it("sends a single selected choice immediately", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <PlanningSessionSurveyForm
        survey={{ questions: [{ prompt: "What is the priority?", options: ["High", "Low", "Medium"] }] }}
        onSubmit={onSubmit}
      />,
    );

    const skip = screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey });
    expect(screen.getByTestId("create-with-agent-survey-card")).toContainElement(skip);
    expect(within(screen.getByRole("button", { name: "High" })).getByText("A")).toBeInTheDocument();
    expect(within(skip).getByText("D")).toBeInTheDocument();
    expect(screen.getByPlaceholderText(CREATE_WITH_AGENT_COPY.otherAnswer)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswer })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /High/ }));

    expect(onSubmit).toHaveBeenCalledWith("What is the priority? High");
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("sends a single skipped question immediately", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={{ questions: [twoQuestions.questions[0]] }} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey }));

    expect(onSubmit).toHaveBeenCalledWith(CREATE_WITH_AGENT_COPY.surveySkipped);
    expect(onSubmit).toHaveBeenCalledTimes(1);
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
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswer })).toBeInTheDocument();
  });

  it("uses Enter on a middle question as Next, not Submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.type(screen.getByRole("textbox"), "Custom{Enter}");

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
  });

  it("keeps a skipped first question when browsing back and sends it with the final answer", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey }));
    expect(screen.getByText("Question 2 of 2")).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion }));
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));
    await user.click(screen.getByRole("button", { name: /One file/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers }));

    expect(onSubmit).toHaveBeenCalledWith("What is the priority? skipped\nWhat is the scope? One file");
  });

  it("sends answered and skipped questions after Skip on the last page", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey }));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers }));

    expect(onSubmit).toHaveBeenCalledWith("What is the priority? High\nWhat is the scope? skipped");
  });

  it("sends every answer after the last question is picked", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: /One file/ }));
    expect(screen.getByText("Question 2 of 2")).toBeInTheDocument();
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
