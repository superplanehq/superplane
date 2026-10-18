import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { ReviewBotPicker } from "./ReviewBotPicker";
import { normalizeReviewBotLogin, reviewBotRows } from "./reviewBotPickerModel";

describe("normalizeReviewBotLogin", () => {
  it("trims, strips @, and drops a trailing [bot]", () => {
    expect(normalizeReviewBotLogin(" @CodeRabbitAI[bot] ")).toBe("CodeRabbitAI");
    expect(normalizeReviewBotLogin("bugbot")).toBe("bugbot");
    expect(normalizeReviewBotLogin("   ")).toBe("");
  });
});

describe("reviewBotRows", () => {
  it("puts selected bots first and then the rest of the catalog", () => {
    expect(
      reviewBotRows(
        [
          { login: "coderabbitai", displayName: "coderabbitai[bot]" },
          { login: "bugbot", displayName: "bugbot[bot]" },
          { login: "graphite", displayName: "graphite[bot]" },
        ],
        ["bugbot", "manual-bot"],
      ),
    ).toEqual([
      { login: "bugbot", displayName: "bugbot[bot]" },
      { login: "manual-bot", displayName: "manual-bot" },
      { login: "coderabbitai", displayName: "coderabbitai[bot]" },
      { login: "graphite", displayName: "graphite[bot]" },
    ]);
  });
});

describe("ReviewBotPicker", () => {
  it("explains found bots and keeps the add field hidden until the user opens it", async () => {
    const user = userEvent.setup();
    render(
      <ReviewBotPicker
        selected={["coderabbitai"]}
        catalog={[{ login: "coderabbitai", displayName: "coderabbitai[bot]" }]}
        onToggle={vi.fn()}
        onAdd={vi.fn().mockReturnValue(true)}
      />,
    );

    expect(screen.getByTestId("discussion-setup-bots-found")).toHaveTextContent(
      "SuperPlane scanned the most recent pull requests in the repository and found these bots.",
    );
    expect(screen.getByTestId("discussion-setup-bots-missing")).toHaveTextContent("The bot that you use is not here?");
    expect(screen.queryByTestId("discussion-setup-bot-manual")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("discussion-setup-bot-add-manually"));
    expect(screen.getByTestId("discussion-setup-bot-manual")).toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bot-add-manually")).not.toBeInTheDocument();
  });

  it("lets the user add a bot login when the catalog is empty", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockReturnValue(true);
    const onToggle = vi.fn();
    render(<ReviewBotPicker selected={[]} catalog={[]} onToggle={onToggle} onAdd={onAdd} />);

    expect(screen.getByTestId("discussion-setup-bots-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bots-found")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("discussion-setup-bot-add-manually"));
    await user.type(screen.getByTestId("discussion-setup-bot-manual"), "coderabbitai");
    await user.click(screen.getByTestId("discussion-setup-bot-add"));
    expect(onAdd).toHaveBeenCalledWith("coderabbitai");
  });

  it("keeps a manually selected bot visible in the list", () => {
    render(
      <ReviewBotPicker
        selected={["coderabbitai"]}
        catalog={[]}
        onToggle={vi.fn()}
        onAdd={vi.fn().mockReturnValue(true)}
      />,
    );

    expect(screen.queryByTestId("discussion-setup-bots-empty")).not.toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-bot-coderabbitai")).toHaveAttribute("aria-selected", "true");
  });

  it("shows a name field immediately when there is no review bot catalog", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockReturnValue(true);
    const onToggle = vi.fn();
    const { rerender } = render(
      <ReviewBotPicker
        selected={[]}
        catalog={[{ login: "bugbot", displayName: "bugbot[bot]" }]}
        loading
        loadError
        supportsReviewBots={false}
        onToggle={onToggle}
        onAdd={onAdd}
      />,
    );

    expect(screen.queryByTestId("discussion-setup-bots-empty")).not.toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bots-missing")).not.toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bots-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bots-loading")).not.toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bots-found")).not.toBeInTheDocument();
    expect(screen.queryByText("SuperPlane could not load review bots from this repository.")).not.toBeInTheDocument();
    expect(screen.getByText("Type the name of each review bot SuperPlane should respond to.")).toBeInTheDocument();
    expect(screen.getByLabelText("Bot name")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Bot name, for example coderabbitai")).toBeInTheDocument();

    await user.type(screen.getByTestId("discussion-setup-bot-manual"), "coderabbitai");
    await user.click(screen.getByTestId("discussion-setup-bot-add"));
    expect(onAdd).toHaveBeenCalledWith("coderabbitai");
    expect(screen.getByTestId("discussion-setup-bot-manual")).toBeInTheDocument();

    rerender(
      <ReviewBotPicker
        selected={["coderabbitai"]}
        catalog={[]}
        supportsReviewBots={false}
        onToggle={onToggle}
        onAdd={onAdd}
      />,
    );

    expect(screen.queryByTestId("discussion-setup-bots-empty")).not.toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bots-missing")).not.toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-bot-manual")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Remove coderabbitai" }));
    expect(onToggle).toHaveBeenCalledWith("coderabbitai");
  });
});
