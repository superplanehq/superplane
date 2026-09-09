import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ReviewBotPicker, normalizeReviewBotLogin, reviewBotRows } from "./ReviewBotPicker";

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
  it("lets the user add a bot login when the catalog is empty", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn().mockReturnValue(true);
    const onToggle = vi.fn();
    render(<ReviewBotPicker selected={[]} catalog={[]} onToggle={onToggle} onAdd={onAdd} />);

    expect(screen.getByTestId("discussion-setup-bots-empty")).toBeInTheDocument();
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
});
