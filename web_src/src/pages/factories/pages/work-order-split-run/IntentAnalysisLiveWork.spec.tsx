import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import { AnalysisLiveWork } from "./IntentAnalysisLiveWork";

describe("AnalysisLiveWork", () => {
  it("shows a two-line reasoning window when lines exist", () => {
    render(
      <AnalysisLiveWork
        machineStatus="running"
        items={[
          { id: "read", text: "Read billing.ts" },
          { id: "note", text: "The empty view only names the page." },
        ]}
      />,
    );

    const stream = screen.getByTestId("split-run-intent-reasoning");
    expect(stream).toHaveTextContent("Read billing.ts");
    expect(stream).toHaveTextContent("The empty view only names the page.");
    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();
  });

  it("shimmers the latest reasoning line while the machine is on", () => {
    render(
      <AnalysisLiveWork
        machineStatus="running"
        items={[
          { id: "read", text: "Read billing.ts" },
          { id: "note", text: "The empty view only names the page." },
        ]}
      />,
    );

    expect(screen.getByTestId("split-run-intent-reasoning-read")).not.toHaveClass("sp-ai-thinking");
    expect(screen.getByTestId("split-run-intent-reasoning-note")).toHaveClass("sp-ai-thinking");
    expect(screen.getByTestId("split-run-intent-reasoning-note")).toHaveAttribute(
      "data-text",
      "The empty view only names the page.",
    );
  });

  it("expands a ran-commands summary to the command names", async () => {
    const user = userEvent.setup();
    render(
      <AnalysisLiveWork
        machineStatus="running"
        items={[{ id: "tools", text: "Ran 2 commands", details: ["ls src", "npm test"] }]}
      />,
    );

    expect(screen.queryByText("ls src")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Ran 2 commands" }));
    expect(screen.getByText("ls src")).toBeInTheDocument();
    expect(screen.getByText("npm test")).toBeInTheDocument();
  });

  it("hides after the machine waits", () => {
    render(<AnalysisLiveWork machineStatus="waiting" items={[{ id: "read", text: "Read billing.ts" }]} />);
    expect(screen.queryByTestId("split-run-intent-reasoning")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();
  });
});
