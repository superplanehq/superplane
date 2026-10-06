import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FirstRunShell } from "./FirstRunShell";

describe("FirstRunShell", () => {
  it("keeps owner setup on the app shell and does not load the art script", () => {
    const scriptsBefore = document.querySelectorAll("script[data-onboarding-src]").length;
    render(
      <FirstRunShell testId="owner-setup" chrome={{ stepIndex: 0, stepCount: 2 }}>
        <p>Create the owner account</p>
      </FirstRunShell>,
    );

    expect(screen.getByTestId("owner-setup")).toHaveAttribute("data-visual", "app");
    expect(screen.queryByTestId("first-run-logo")).not.toBeInTheDocument();
    expect(document.querySelectorAll("script[data-onboarding-src]")).toHaveLength(scriptsBefore);
  });
});
