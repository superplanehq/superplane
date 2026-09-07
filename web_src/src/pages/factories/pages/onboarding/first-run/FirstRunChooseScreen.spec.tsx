import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";

describe("FirstRunChooseScreen", () => {
  it("shows an access hint next to the edit connection link", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-choose-access-hint")).toHaveTextContent(FIRST_RUN_COPY.choose.accessHint);
  });

  it("calls onEditConnection when the edit connection link is clicked", async () => {
    const user = userEvent.setup();
    const onEditConnection = vi.fn();

    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={onEditConnection}
        onContinue={vi.fn()}
      />,
    );

    await user.click(screen.getByText(FIRST_RUN_COPY.choose.editConnection));
    expect(onEditConnection).toHaveBeenCalled();
  });

  it("reveals why a repository might not show up when expanded", async () => {
    const user = userEvent.setup();

    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    const disclosure = screen.getByTestId("first-run-choose-why-missing");
    expect(screen.getByText(FIRST_RUN_COPY.choose.missingTitle)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.choose.missingReasons[0])).not.toBeVisible();

    await user.click(screen.getByText(FIRST_RUN_COPY.choose.missingTitle));

    expect(disclosure).toHaveAttribute("open");
    for (const reason of FIRST_RUN_COPY.choose.missingReasons) {
      expect(screen.getByText(reason)).toBeVisible();
    }
  });

  it("disables continue until a repository is selected", () => {
    render(
      <FirstRunChooseScreen
        repositories={["octo/repo"]}
        selectedRepository={null}
        onSelectRepository={vi.fn()}
        onEditConnection={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeDisabled();
  });
});
