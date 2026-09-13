import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
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

  it("hides the starting status after the first agent line", () => {
    render(
      <AnalysisLiveWork
        machineStatus="running"
        waitingForAgent
        items={[
          { id: "note", text: "I have enough understanding of the repository. Let me write the analysis files." },
          { id: "tools", text: "Ran 1 command", details: ["jq empty /tmp/intake-analysis.json"] },
        ]}
      />,
    );

    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-reasoning")).toHaveTextContent(
      "I have enough understanding of the repository.",
    );
  });

  it("keeps the starting status while commands run before the first agent message", () => {
    render(
      <AnalysisLiveWork
        machineStatus="running"
        waitingForAgent
        items={[{ id: "tools", text: "Ran 2 commands", details: ["git clone", "ls"] }]}
      />,
    );

    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent(CREATE_WITH_AGENT_COPY.machineStarting);
    expect(screen.getByRole("button", { name: "Ran 2 commands" })).toBeInTheDocument();
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
