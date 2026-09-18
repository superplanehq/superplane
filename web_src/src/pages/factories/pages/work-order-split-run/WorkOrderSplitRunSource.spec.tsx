import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import type { SplitRunSource } from "./splitRunSource";
import { WorkOrderSplitRunSource } from "./WorkOrderSplitRunSource";

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

describe("WorkOrderSplitRunSource", () => {
  it("tints a monochrome compact source logo to the bubble text color", () => {
    render(<WorkOrderSplitRunSource compact source={GITHUB_SOURCE} />);

    expect(screen.getByRole("img", { name: "GitHub" })).toHaveClass("brightness-0", "invert", "dark:invert-0");
  });

  it("tints a SuperPlane compact source logo to the bubble text color", () => {
    render(<WorkOrderSplitRunSource compact source={SUPERPLANE_SOURCE} />);

    expect(screen.getByRole("img", { name: "SuperPlane" })).toHaveClass("brightness-0", "invert", "dark:invert-0");
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
});
