import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { HumanReviewPolicySettings } from "./HumanReviewPolicySettings";

describe("HumanReviewPolicySettings", () => {
  it("enables review with an individual default and preserves the entrypoint configuration", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(
      <HumanReviewPolicySettings
        node={{
          id: "on-human-review",
          name: "Human review",
          configuration: { enabled: false, fallbackTeam: "", parameters: { pullRequestId: "required" } },
        }}
        reviewers={["mytmlt", "maintainer"]}
        onSave={onSave}
      />,
    );

    await user.click(screen.getByRole("switch", { name: "Enable Human review" }));
    expect(screen.getByRole("button", { name: "Save Human review settings" })).toBeDisabled();
    fireEvent.click(screen.getByRole("combobox", { name: "Default reviewer" }));
    fireEvent.click(await screen.findByRole("option", { name: "@mytmlt" }));
    await user.click(screen.getByRole("button", { name: "Save Human review settings" }));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith(
        expect.objectContaining({
          nodeId: "on-human-review",
          configuration: {
            enabled: true,
            defaultReviewer: "mytmlt",
            fallbackTeam: "",
            parameters: { pullRequestId: "required" },
          },
        }),
      ),
    );
  });

  it("preserves a saved reviewer when the reviewer list fails and supports retry", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const onRetryReviewers = vi.fn();
    const user = userEvent.setup();
    render(
      <HumanReviewPolicySettings
        node={{ id: "on-human-review", configuration: { enabled: true, defaultReviewer: "mytmlt" } }}
        reviewers={[]}
        reviewersError="SuperPlane could not load reviewers. Check GitHub App access and try again."
        onRetryReviewers={onRetryReviewers}
        onSave={onSave}
      />,
    );

    expect(screen.getByRole("combobox", { name: "Default reviewer" })).toHaveTextContent("@mytmlt");
    expect(screen.getByRole("alert")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetryReviewers).toHaveBeenCalledTimes(1);
    await user.click(screen.getByRole("button", { name: "Save Human review settings" }));
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({ configuration: { enabled: true, defaultReviewer: "mytmlt", fallbackTeam: "" } }),
    );
  });

  it("keeps an existing fallback team available without a default reviewer", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(
      <HumanReviewPolicySettings
        node={{ id: "on-human-review", configuration: { enabled: true, fallbackTeam: "acme/reviewers" } }}
        reviewers={[]}
        onSave={onSave}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Save Human review settings" }));
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        configuration: { enabled: true, defaultReviewer: "", fallbackTeam: "acme/reviewers" },
      }),
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
      expect.objectContaining({ configuration: { enabled: false, defaultReviewer: "", fallbackTeam: "" } }),
    );
  });
});
