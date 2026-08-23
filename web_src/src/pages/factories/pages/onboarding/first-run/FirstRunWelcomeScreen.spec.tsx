import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FIRST_RUN_PREVIEW_TICKETS, FIRST_RUN_STORY_EMAIL, firstRunStoryChrome } from "./firstRunMocks";
import { FirstRunWelcomeScreen } from "./FirstRunWelcomeScreen";

describe("FirstRunWelcomeScreen", () => {
  it("shows the greeting, chrome, preview, and Get started", async () => {
    const user = userEvent.setup();
    const onGetStarted = vi.fn();
    const onLogOut = vi.fn();

    render(
      <FirstRunWelcomeScreen
        firstName="Ada"
        previewTickets={FIRST_RUN_PREVIEW_TICKETS}
        chrome={{ ...firstRunStoryChrome(0), onLogOut }}
        onGetStarted={onGetStarted}
      />,
    );

    expect(screen.getByText("Hi Ada.")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: FIRST_RUN_COPY.welcome.headline })).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.welcome.intro)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.connectGitHub)).not.toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.welcome.previewCaption)).toBeInTheDocument();
    expect(screen.getByTestId("first-run-log-out")).toHaveTextContent(FIRST_RUN_COPY.chrome.logOut);
    expect(screen.getByTestId("first-run-signed-in")).toHaveTextContent(FIRST_RUN_STORY_EMAIL);

    await user.click(screen.getByTestId("first-run-get-started"));
    expect(onGetStarted).toHaveBeenCalled();

    await user.click(screen.getByTestId("first-run-log-out"));
    expect(onLogOut).toHaveBeenCalled();
  });
});
