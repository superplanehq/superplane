import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RunErrorsCard } from "./RunErrorsCard";

describe("RunErrorsCard", () => {
  it("renders nothing when there are no errors", () => {
    const { container } = render(<RunErrorsCard errors={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("shows a single run error", () => {
    render(<RunErrorsCard errors={["pipeline failed"]} />);

    expect(screen.getByRole("alert")).toHaveTextContent("This run has an error");
    expect(screen.getByText("pipeline failed")).toBeInTheDocument();
  });

  it("lists multiple run errors", () => {
    render(<RunErrorsCard errors={["pipeline failed", "tests failed"]} />);

    expect(screen.getByRole("alert")).toHaveTextContent("This run has errors");
    expect(screen.getByText("pipeline failed")).toBeInTheDocument();
    expect(screen.getByText("tests failed")).toBeInTheDocument();
  });
});
