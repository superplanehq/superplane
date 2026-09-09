import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SurveyWidget, type SurveyQuestion } from "./SurveyWidget";

const questions: SurveyQuestion[] = [
  { prompt: "What do you want to automate?", options: ["Deploys"], hasInput: true },
  { prompt: "How often does it run?", options: ["Daily"], hasInput: true },
];

describe("SurveyWidget", () => {
  it("advances to the next question when Enter is pressed in the custom answer input", () => {
    render(<SurveyWidget questions={questions} />);

    const input = screen.getByPlaceholderText("Type your own answer...");
    fireEvent.change(input, { target: { value: "Release notes" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(screen.getByText("How often does it run?")).toBeTruthy();
    expect(screen.getByText("2/2")).toBeTruthy();
  });

  it("submits the survey when Enter is pressed on the last question", () => {
    const onAction = vi.fn();
    render(<SurveyWidget questions={questions} onAction={onAction} />);

    const input = screen.getByPlaceholderText("Type your own answer...");
    fireEvent.change(input, { target: { value: "Release notes" } });
    fireEvent.keyDown(input, { key: "Enter" });

    fireEvent.change(screen.getByPlaceholderText("Type your own answer..."), { target: { value: "Every hour" } });
    fireEvent.keyDown(screen.getByPlaceholderText("Type your own answer..."), { key: "Enter" });

    expect(onAction).toHaveBeenCalledWith(
      "What do you want to automate? → Release notes\nHow often does it run? → Every hour",
    );
  });

  it("ignores Enter while an IME composition is in progress", () => {
    render(<SurveyWidget questions={questions} />);

    const input = screen.getByPlaceholderText("Type your own answer...");
    fireEvent.keyDown(input, { key: "Enter", isComposing: true });

    expect(screen.getByText("What do you want to automate?")).toBeTruthy();
    expect(screen.getByText("1/2")).toBeTruthy();
  });
});
