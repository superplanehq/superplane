import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import { CREATE_WITH_AGENT_DEMO_MODELS } from "./createWithAgentDemo";
import { CreateWithAgentModelPicker } from "./CreateWithAgentModelPicker";

describe("CreateWithAgentModelPicker", () => {
  it("shows Using {label} and lists hosted plus BYOK rows", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <CreateWithAgentModelPicker
        models={CREATE_WITH_AGENT_DEMO_MODELS}
        selectedKey="hosted::anthropic::claude-sonnet-4-6"
        selectedLabel="anthropic/claude-sonnet-4-6"
        onSelect={onSelect}
      />,
    );

    expect(screen.getByTestId("create-with-agent-model")).toHaveTextContent(
      CREATE_WITH_AGENT_COPY.usingModel("anthropic/claude-sonnet-4-6"),
    );
    await user.click(screen.getByTestId("create-with-agent-model"));
    const picker = screen.getByTestId("create-with-agent-model-picker");
    expect(picker).toHaveTextContent("SuperPlane");
    expect(picker).toHaveTextContent("Your keys");
    const hosted = screen.getByTestId("create-with-agent-model-option-hosted::anthropic::claude-sonnet-4-6");
    expect(hosted).toHaveTextContent("anthropic/claude-sonnet-4-6");
    expect(hosted).toHaveTextContent("Anthropic");
    expect(hosted).not.toHaveTextContent("SuperPlane");
    expect(hosted).not.toHaveTextContent("Your keys");
    const byok = screen.getByTestId("create-with-agent-model-option-byok::anthropic::claude-sonnet-4-6");
    expect(byok).toHaveTextContent("Anthropic");
    expect(byok).not.toHaveTextContent("Your keys");
    await user.click(byok);
    expect(onSelect).toHaveBeenCalledWith("byok::anthropic::claude-sonnet-4-6");
  });

  it("turns the control off while the machine starts", () => {
    render(
      <CreateWithAgentModelPicker
        models={CREATE_WITH_AGENT_DEMO_MODELS}
        selectedKey="hosted::anthropic::claude-sonnet-4-6"
        selectedLabel="anthropic/claude-sonnet-4-6"
        disabled
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByTestId("create-with-agent-model")).toBeDisabled();
  });
});
