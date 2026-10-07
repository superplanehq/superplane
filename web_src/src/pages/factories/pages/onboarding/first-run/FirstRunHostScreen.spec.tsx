import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { IntegrationChoiceIcon } from "../onboardingSteps";
import { FirstRunHostScreen } from "./FirstRunHostScreen";
import { FirstRunModelSourceChoice } from "./FirstRunModelSourceChoice";
import { FirstRunShell } from "./FirstRunShell";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";

function markIn(container: HTMLElement): HTMLImageElement {
  const image = container.querySelector("img");
  if (!(image instanceof HTMLImageElement)) {
    throw new Error("missing mark");
  }
  return image;
}

function expectLightMark(image: HTMLImageElement) {
  expect(image).toHaveClass("brightness-0", "invert");
}

function expectColoredMark(image: HTMLImageElement) {
  expect(image).not.toHaveClass("brightness-0");
  expect(image).not.toHaveClass("invert");
}

describe("FirstRunHostScreen", () => {
  it("shows a light GitHub mark on the dark host card without a dark ancestor", () => {
    render(<FirstRunHostScreen selectedHost={null} onChooseHost={vi.fn()} />);

    const host = screen.getByTestId("first-run-host");
    expect(host).not.toHaveClass("dark");
    expect(host.closest(".dark")).toBeNull();
    expect(document.documentElement).not.toHaveClass("dark");

    expectLightMark(markIn(screen.getByTestId("first-run-host-github")));
    expectColoredMark(markIn(screen.getByTestId("first-run-host-bitbucket")));
    expectColoredMark(markIn(screen.getByTestId("first-run-host-gitlab")));
    expectColoredMark(screen.getByTestId("first-run-logo"));
  });

  it("shows a light GitHub mark on the ticket choice in the same preview", () => {
    render(<FirstRunTicketsScreen ticketSource={null} onSelectTicketSource={vi.fn()} onAnalyzeTickets={vi.fn()} />);

    const tickets = screen.getByTestId("first-run-tickets");
    expect(tickets.closest(".dark")).toBeNull();
    expectLightMark(markIn(screen.getByRole("button", { name: /GitHub Issues/ })));
    expectColoredMark(markIn(screen.getByRole("button", { name: /Jira/ })));
    expectColoredMark(markIn(screen.getByRole("button", { name: /Linear/ })));
  });

  it("shows a light SuperPlane mark on the hosted model row without waiting for dark", () => {
    render(<FirstRunModelSourceChoice disabled={false} modelSource="hosted" onSelectModelSource={vi.fn()} />);

    expect(document.documentElement).not.toHaveClass("dark");
    expectLightMark(markIn(screen.getByTestId("first-run-model-source")));
  });

  it("inverts other monochrome marks on the preview and leaves colored marks", () => {
    render(
      <FirstRunShell testId="preview-marks" visual="preview">
        <span data-testid="openrouter-mark">
          <IntegrationChoiceIcon name="openrouter" />
        </span>
        <span data-testid="gitlab-mark">
          <IntegrationChoiceIcon name="gitlab" />
        </span>
      </FirstRunShell>,
    );

    expectLightMark(markIn(screen.getByTestId("openrouter-mark")));
    expectColoredMark(markIn(screen.getByTestId("gitlab-mark")));
  });

  it("keeps the GitHub mark dark on a light screen", () => {
    const { container } = render(<IntegrationChoiceIcon name="github" />);

    const image = markIn(container);
    expect(image).toHaveClass("dark:brightness-0", "dark:invert");
    expectColoredMark(image);
  });
});
