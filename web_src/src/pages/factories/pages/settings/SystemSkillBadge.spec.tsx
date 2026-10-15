import { describe, expect, it } from "bun:test";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { SystemSkillBadge } from "./SystemSkillBadge";

describe("SystemSkillBadge", () => {
  it("explains system skills on hover and keyboard focus", async () => {
    const user = userEvent.setup();
    render(<SystemSkillBadge />);
    const badge = screen.getByText("System", { exact: true });
    await user.hover(badge);
    expect(await screen.findByRole("tooltip")).toHaveTextContent(
      "Provided by SuperPlane. You can edit its instructions, but cannot rename, disable, or delete it.",
    );
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument());
    await user.tab();
    expect(badge).toHaveFocus();
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Provided by SuperPlane.");
  });
});
