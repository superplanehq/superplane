import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";

import sentryIcon from "@/assets/icons/integrations/sentry.svg";

import type { SplitRunSource } from "./splitRunSource";
import { WorkOrderSplitRunSource } from "./WorkOrderSplitRunSource";

const AUTOMATION_SOURCE: SplitRunSource = {
  kind: "automation",
  name: "ESLint AutoFix",
  iconSrc: "/superplane.svg",
  iconAlt: "SuperPlane",
  run: { appId: "canvas-eslint", runId: "run-eslint-1" },
};

function renderInFactory(source: SplitRunSource, compact = false) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/tasks/1"]}>
      <Routes>
        <Route
          path="/:organizationId/workspaces/:factoryKey/*"
          element={<WorkOrderSplitRunSource compact={compact} source={source} />}
        />
      </Routes>
    </MemoryRouter>,
  );
}

const GITHUB_SOURCE: SplitRunSource = {
  kind: "intake",
  name: "GitHub issues",
  iconSrc: "/github.svg",
  iconAlt: "GitHub",
  ticket: { label: "acme/payments-service#842", href: "https://github.com/acme/payments-service/issues/842" },
};

const JIRA_SOURCE: SplitRunSource = {
  kind: "intake",
  name: "Jira issues",
  iconSrc: "/jira.svg",
  iconAlt: "Jira",
  ticket: { label: "PAY-12", href: "https://acme.atlassian.net/browse/PAY-12" },
};

const SUPERPLANE_SOURCE: SplitRunSource = {
  kind: "intake",
  name: "SuperPlane",
  iconSrc: "/superplane.svg",
  iconAlt: "SuperPlane",
};

const SENTRY_SOURCE: SplitRunSource = {
  kind: "intake",
  name: "Sentry exceptions",
  iconSrc: sentryIcon,
  iconAlt: "Sentry",
};

describe("WorkOrderSplitRunSource", () => {
  it("tints a monochrome compact source logo to the bubble text color", () => {
    render(<WorkOrderSplitRunSource compact source={GITHUB_SOURCE} />);

    expect(screen.getByRole("img", { name: "GitHub" })).toHaveClass("dark:brightness-0", "dark:invert");
  });

  it("tints a SuperPlane compact source logo to the bubble text color", () => {
    render(<WorkOrderSplitRunSource compact source={SUPERPLANE_SOURCE} />);

    expect(screen.getByRole("img", { name: "SuperPlane" })).toHaveClass("dark:brightness-0", "dark:invert");
  });

  it("does not tint a colored compact source logo", () => {
    render(<WorkOrderSplitRunSource compact source={JIRA_SOURCE} />);

    expect(screen.getByRole("img", { name: "Jira" })).not.toHaveClass("brightness-0");
    expect(screen.getByRole("img", { name: "Jira" })).not.toHaveClass("invert");
    expect(screen.getByRole("img", { name: "Jira" })).not.toHaveClass("dark:invert-0");
  });

  it("tints a monochrome card source logo in dark mode", () => {
    render(<WorkOrderSplitRunSource source={GITHUB_SOURCE} />);

    expect(screen.getByRole("img", { name: "GitHub" })).toHaveClass("dark:brightness-0", "dark:invert");
  });

  it("does not tint a colored card source logo", () => {
    render(<WorkOrderSplitRunSource source={JIRA_SOURCE} />);

    expect(screen.getByRole("img", { name: "Jira" })).not.toHaveClass("dark:brightness-0");
    expect(screen.getByRole("img", { name: "Jira" })).not.toHaveClass("dark:invert");
  });

  it("inverts a compact Sentry source logo in dark mode without bubble tint", () => {
    render(<WorkOrderSplitRunSource compact source={SENTRY_SOURCE} />);

    const icon = screen.getByRole("img", { name: "Sentry" });
    expect(icon).toHaveClass("dark:brightness-0", "dark:invert");
    expect(icon).not.toHaveClass("brightness-0");
    expect(icon).not.toHaveClass("invert");
    expect(icon).not.toHaveClass("dark:invert-0");
  });

  it("inverts a Sentry card source logo in dark mode", () => {
    render(<WorkOrderSplitRunSource source={SENTRY_SOURCE} />);

    expect(screen.getByRole("img", { name: "Sentry" })).toHaveClass("dark:brightness-0", "dark:invert");
  });

  it("names an automation source with its name and the SuperPlane mark", () => {
    render(<WorkOrderSplitRunSource source={AUTOMATION_SOURCE} />);

    expect(screen.getByText("ESLint AutoFix")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "SuperPlane" })).toBeInTheDocument();
  });

  for (const compact of [false, true]) {
    it(`links the task to the run that created it (compact: ${compact})`, () => {
      renderInFactory(AUTOMATION_SOURCE, compact);

      const link = screen.getByRole("link", { name: "View run" });
      expect(link.getAttribute("href")).toContain("canvas-eslint");
      expect(link.getAttribute("href")).toContain("run-eslint-1");
    });

    it(`shows no run link when the task has no source run (compact: ${compact})`, () => {
      renderInFactory({ ...AUTOMATION_SOURCE, run: undefined }, compact);

      expect(screen.getByText("ESLint AutoFix")).toBeInTheDocument();
      expect(screen.queryByRole("link", { name: "View run" })).not.toBeInTheDocument();
    });
  }
});
