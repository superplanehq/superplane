import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { firstRunStoryChrome } from "./firstRunMocks";
import { FirstRunWelcomeScreen } from "./FirstRunWelcomeScreen";

describe("FirstRunWelcomeScreen", () => {
  it("shows the greeting, chrome, and Get started", async () => {
    const user = userEvent.setup();
    const onGetStarted = vi.fn();
    const onQuitOnboarding = vi.fn();

    render(
      <MemoryRouter>
        <FirstRunWelcomeScreen
          firstName="Ada"
          chrome={{ ...firstRunStoryChrome(0), onQuitOnboarding }}
          onGetStarted={onGetStarted}
        />
      </MemoryRouter>,
    );

    expect(screen.getByText("Hi Ada.")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: FIRST_RUN_COPY.welcome.headline })).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.welcome.intro)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.connectGitHub)).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-ticket-list")).not.toBeInTheDocument();
    expect(screen.queryByText("Example: tickets scored from a real backlog.")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-get-started"));
    expect(onGetStarted).toHaveBeenCalled();

    await user.click(screen.getByTestId("factories-sidebar-user-menu-trigger"));
    await user.click(screen.getByRole("menuitem", { name: "Quit onboarding" }));
    expect(onQuitOnboarding).toHaveBeenCalled();
  });
});
