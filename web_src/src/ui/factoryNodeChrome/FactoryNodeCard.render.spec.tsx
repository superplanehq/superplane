import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FactoryNodeCard } from "./FactoryNodeCard";
import { FactoryNodeStepList } from "./FactoryNodeStepList";

describe("FactoryNodeCard", () => {
  it("shows configured steps in a wider node body", () => {
    render(
      <FactoryNodeCard
        title="Run Claude Code"
        componentLabel="Run Claude Code"
        nodeName="Agent - No GH Issue Plan"
        iconSlug="code"
        canvasMode="edit"
        body={<FactoryNodeStepList steps={["Clone repo", "Write implementation plan", "Use plan as output"]} />}
      />,
    );

    const card = screen.getByTestId("factory-node-run-claude-code");
    expect(card).toHaveStyle({ width: "320px" });
    expect(screen.getByText("Clone repo")).toBeInTheDocument();
    expect(screen.getByText("Write implementation plan")).toBeInTheDocument();
    expect(screen.getByText("Use plan as output")).toBeInTheDocument();
  });

  it("keeps ordinary factory nodes at the default width", () => {
    render(<FactoryNodeCard title="Run Bash" componentLabel="Run Bash" nodeName="Check files" />);

    expect(screen.getByTestId("factory-node-run-bash")).toHaveStyle({ width: "280px" });
  });
});
