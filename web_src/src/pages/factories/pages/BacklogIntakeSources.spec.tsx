import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { BacklogIntakeSources } from "./BacklogIntakeSources";
import { DEFAULT_GITHUB_INTAKE_SETTINGS } from "./intakeSourceSettingsModel";
import { lineIntakeSourceById, type ConfiguredLineIntakeSource } from "./lineIntakeModel";

function configuredIntake(overrides: Partial<ConfiguredLineIntakeSource> = {}): ConfiguredLineIntakeSource {
  return {
    intakeId: "intake-github",
    appId: "app-github-issues-intake",
    healthy: true,
    settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS },
    source: lineIntakeSourceById("github-issues")!,
    ...overrides,
  };
}

const GITHUB_INTAKE = configuredIntake();

const SENTRY_INTAKE = configuredIntake({
  intakeId: "intake-sentry",
  appId: "app-sentry-intake",
  source: lineIntakeSourceById("sentry-exceptions")!,
});

function renderSources(props: Partial<Parameters<typeof BacklogIntakeSources>[0]> = {}) {
  return render(
    <BacklogIntakeSources
      intakes={props.intakes ?? [GITHUB_INTAKE, SENTRY_INTAKE]}
      showAddIntake={props.showAddIntake}
      onOpenSettings={props.onOpenSettings ?? vi.fn()}
      onAddIntake={props.onAddIntake ?? vi.fn()}
    />,
  );
}

describe("BacklogIntakeSources", () => {
  it("titles each intake with what it listens to, and nothing else", () => {
    renderSources({ intakes: [GITHUB_INTAKE] });

    const intake = screen.getByTestId("line-intake-source-intake-github");
    expect(intake).toHaveTextContent("Listening to GitHub issues");
    expect(intake).not.toHaveTextContent("Creates tasks from GitHub issues.");
    expect(screen.queryByText("Automations that listen, evaluate, and create backlog work orders.")).toBeNull();
  });

  it("lists two intakes on the same source as separate rows", () => {
    renderSources({
      intakes: [
        GITHUB_INTAKE,
        configuredIntake({
          intakeId: "intake-github-triage",
          appId: "app-triage",
          source: { ...lineIntakeSourceById("github-issues")!, name: "Triage issues" },
        }),
      ],
    });

    expect(screen.getByTestId("line-intake-source-intake-github")).toHaveTextContent("Listening to GitHub issues");
    expect(screen.getByTestId("line-intake-source-intake-github-triage")).toHaveTextContent(
      "Listening to Triage issues",
    );
  });

  it("shows only the intakes it is given", () => {
    renderSources({ intakes: [GITHUB_INTAKE] });

    expect(screen.getByTestId("line-intake-source-intake-github")).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-source-intake-sentry")).not.toBeInTheDocument();
  });

  it("marks an intake whose automation can no longer create work orders", () => {
    renderSources({ intakes: [configuredIntake({ healthy: false })] });

    const intake = screen.getByTestId("line-intake-source-intake-github");
    expect(within(intake).getByTestId("line-intake-source-intake-github-needs-repair")).toHaveTextContent(
      "Needs repair",
    );
  });

  it("opens the settings of the intake the user clicked", async () => {
    const onOpenSettings = vi.fn();
    const user = userEvent.setup();
    renderSources({ onOpenSettings });

    await user.click(screen.getByRole("button", { name: "Open Sentry exceptions settings" }));

    expect(onOpenSettings).toHaveBeenCalledWith(SENTRY_INTAKE);
  });

  it("hides the Add intake control by default", () => {
    renderSources();

    expect(screen.queryByTestId("line-intake-add")).not.toBeInTheDocument();
  });

  it("renders nothing when the workspace declared no intake", () => {
    renderSources({ intakes: [] });

    expect(screen.queryByTestId("lines-backlog-intakes")).not.toBeInTheDocument();
  });

  it("offers Add intake when the caller enables it", async () => {
    const onAddIntake = vi.fn();
    const user = userEvent.setup();
    renderSources({ showAddIntake: true, onAddIntake });

    await user.click(screen.getByTestId("line-intake-add"));

    expect(onAddIntake).toHaveBeenCalledTimes(1);
  });
});
