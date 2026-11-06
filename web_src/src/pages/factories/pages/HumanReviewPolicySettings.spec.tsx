import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { HumanReviewPolicySettings } from "./HumanReviewPolicySettings";

describe("HumanReviewPolicySettings", () => {
  it("requires a fallback team before saving enablement and preserves the entrypoint configuration", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(
      <HumanReviewPolicySettings
        node={{
          id: "on-human-review",
          name: "Human review",
          configuration: { enabled: false, fallbackTeam: "", parameters: { pullRequestId: "required" } },
        }}
        onSave={onSave}
      />,
    );

    await user.click(screen.getByRole("switch", { name: "Enable Human review" }));
    expect(screen.getByRole("button", { name: "Save Human review settings" })).toBeDisabled();
    await user.type(screen.getByRole("textbox", { name: "Fallback GitHub team" }), "acme/reviewers");
    await user.click(screen.getByRole("button", { name: "Save Human review settings" }));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith(
        expect.objectContaining({
          nodeId: "on-human-review",
          configuration: { enabled: true, fallbackTeam: "acme/reviewers", parameters: { pullRequestId: "required" } },
        }),
      ),
    );
  });

  it("allows disablement without a fallback and shows a server rejection", async () => {
    const onSave = vi.fn().mockRejectedValue(new Error("Only organization administrators can update Human review."));
    const user = userEvent.setup();
    render(
      <HumanReviewPolicySettings node={{ id: "on-human-review", configuration: { enabled: false } }} onSave={onSave} />,
    );

    await user.click(screen.getByRole("button", { name: "Save Human review settings" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Only organization administrators");
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({ configuration: { enabled: false, fallbackTeam: "" } }),
    );
  });
});
