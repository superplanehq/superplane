import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { IntegrationStatusIndicator, type MissingIntegration } from "./";

const githubMissing: MissingIntegration = {
  integrationName: "github",
  affectedNodeCount: 3,
};

const slackMissing: MissingIntegration = {
  integrationName: "slack",
  affectedNodeCount: 1,
};

const headerButton = () => screen.getByRole("button", { name: /integration/ });

describe("IntegrationStatusIndicator", () => {
  it("renders nothing when there are no missing integrations", () => {
    const { container } = render(<IntegrationStatusIndicator missingIntegrations={[]} onConnect={vi.fn()} />);

    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing when every missing integration has just been connected", () => {
    const { container } = render(
      <IntegrationStatusIndicator
        missingIntegrations={[{ ...githubMissing, justConnected: true }]}
        onConnect={vi.fn()}
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("counts only integrations that have not just been connected", () => {
    render(
      <IntegrationStatusIndicator
        missingIntegrations={[{ ...githubMissing, justConnected: true }, slackMissing]}
        onConnect={vi.fn()}
      />,
    );

    expect(headerButton()).toHaveTextContent("1 integration needs setup");
    expect(screen.getByText("GitHub")).toBeInTheDocument();
    expect(screen.getByText("Slack")).toBeInTheDocument();
  });

  it("pluralizes the header when more than one integration needs setup", () => {
    render(<IntegrationStatusIndicator missingIntegrations={[githubMissing, slackMissing]} onConnect={vi.fn()} />);

    expect(headerButton()).toHaveTextContent("2 integrations need setup");
  });

  it("shows the known display name rather than the raw integration name", () => {
    render(<IntegrationStatusIndicator missingIntegrations={[githubMissing]} onConnect={vi.fn()} />);

    expect(screen.getByText("GitHub")).toBeInTheDocument();
    expect(screen.queryByText("github")).not.toBeInTheDocument();
  });

  it("falls back to the integration name when there is no known display name", () => {
    render(
      <IntegrationStatusIndicator
        missingIntegrations={[{ integrationName: "acme-ci", affectedNodeCount: 1 }]}
        onConnect={vi.fn()}
      />,
    );

    expect(screen.getByText("Acme-ci")).toBeInTheDocument();
  });

  it("pluralizes the affected node count per integration", () => {
    render(<IntegrationStatusIndicator missingIntegrations={[githubMissing, slackMissing]} onConnect={vi.fn()} />);

    expect(screen.getByText("3 nodes")).toBeInTheDocument();
    expect(screen.getByText("1 node")).toBeInTheDocument();
  });

  it("calls onConnect with the integration name when Connect is clicked", async () => {
    const user = userEvent.setup();
    const onConnect = vi.fn();

    render(<IntegrationStatusIndicator missingIntegrations={[githubMissing]} onConnect={onConnect} />);

    await user.click(screen.getByRole("button", { name: "Connect" }));

    expect(onConnect).toHaveBeenCalledTimes(1);
    expect(onConnect).toHaveBeenCalledWith("github");
  });

  it("disables the Connect button in read-only mode", () => {
    render(<IntegrationStatusIndicator missingIntegrations={[githubMissing]} onConnect={vi.fn()} readOnly />);

    expect(screen.getByRole("button", { name: "Connect" })).toBeDisabled();
  });

  it("disables the Connect button when the user cannot create integrations", () => {
    render(
      <IntegrationStatusIndicator
        missingIntegrations={[githubMissing]}
        onConnect={vi.fn()}
        canCreateIntegrations={false}
      />,
    );

    expect(screen.getByRole("button", { name: "Connect" })).toBeDisabled();
  });

  it("offers Configure instead of Connect when an integration instance already exists", () => {
    render(
      <IntegrationStatusIndicator missingIntegrations={[{ ...githubMissing, state: "error" }]} onConnect={vi.fn()} />,
    );

    expect(screen.getByRole("button", { name: "Configure" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect" })).not.toBeInTheDocument();
  });

  it("renders a state badge for pending and error integrations", () => {
    const { unmount } = render(
      <IntegrationStatusIndicator missingIntegrations={[{ ...githubMissing, state: "pending" }]} onConnect={vi.fn()} />,
    );

    expect(screen.getByText("Pending")).toBeInTheDocument();
    unmount();

    render(
      <IntegrationStatusIndicator missingIntegrations={[{ ...githubMissing, state: "error" }]} onConnect={vi.fn()} />,
    );

    expect(screen.getByText("Error")).toBeInTheDocument();
  });

  it("marks the state badge as help-cursor only when a description explains it", () => {
    const { unmount } = render(
      <IntegrationStatusIndicator missingIntegrations={[{ ...githubMissing, state: "error" }]} onConnect={vi.fn()} />,
    );

    expect(screen.getByText("Error")).not.toHaveClass("cursor-help");
    unmount();

    render(
      <IntegrationStatusIndicator
        missingIntegrations={[{ ...githubMissing, state: "error", stateDescription: "Token expired" }]}
        onConnect={vi.fn()}
      />,
    );

    expect(screen.getByText("Error")).toHaveClass("cursor-help");
  });

  it("replaces the action button with a Connected label for just-connected integrations", () => {
    render(
      <IntegrationStatusIndicator
        missingIntegrations={[{ ...githubMissing, justConnected: true }, slackMissing]}
        onConnect={vi.fn()}
      />,
    );

    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Connect" })).toHaveLength(1);
  });

  it("collapses to a summary and expands again when the header is toggled", async () => {
    const user = userEvent.setup();

    render(<IntegrationStatusIndicator missingIntegrations={[githubMissing, slackMissing]} onConnect={vi.fn()} />);

    await user.click(headerButton());

    expect(headerButton()).toHaveTextContent("2 integrations");
    expect(screen.queryByRole("button", { name: "Connect" })).not.toBeInTheDocument();

    await user.click(headerButton());

    expect(headerButton()).toHaveTextContent("2 integrations need setup");
    expect(screen.getAllByRole("button", { name: "Connect" })).toHaveLength(2);
  });

  it("uses the singular noun in the collapsed summary for a single integration", async () => {
    const user = userEvent.setup();

    render(<IntegrationStatusIndicator missingIntegrations={[githubMissing]} onConnect={vi.fn()} />);

    await user.click(headerButton());

    expect(headerButton()).toHaveTextContent("1 integration");
  });
});