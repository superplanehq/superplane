import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

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
    render(<PlanningSessionSurveyForm survey={twoQuestions} onSubmit={vi.fn()} />);

    expect(screen.getByText("What is the priority?")).toBeInTheDocument();
    expect(screen.queryByText("What is the scope?")).not.toBeInTheDocument();
    expect(screen.getByText("1 of 2")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));

    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
    expect(screen.queryByText("What is the priority?")).not.toBeInTheDocument();
    expect(screen.getByText("2 of 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.previousQuestion }));

    expect(screen.getByText("What is the priority?")).toBeInTheDocument();
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
