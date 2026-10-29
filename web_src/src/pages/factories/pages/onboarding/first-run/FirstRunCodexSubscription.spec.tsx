import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunCodexSubscription } from "./FirstRunCodexSubscription";

const copy = FIRST_RUN_COPY.agent;

function renderPanel(
  connection: {
    status?: string;
    verificationUrl?: string;
    verificationCode?: string;
    accountLabel?: string;
    reservationHeld?: boolean;
  },
  handlers: { onStart?: () => void; onCancel?: () => void } = {},
) {
  return render(
    <FirstRunCodexSubscription
      selected
      disabled={false}
      starting={false}
      canceling={false}
      connection={connection}
      onSelect={vi.fn()}
      onStart={handlers.onStart ?? vi.fn()}
      onCancel={handlers.onCancel ?? vi.fn()}
    />,
  );
}

describe("FirstRunCodexSubscription", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows the device link and copies the sign-in code", async () => {
    const user = userEvent.setup();
    const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
    renderPanel({
      status: "pending",
      verificationUrl: "https://auth.openai.com/codex/device",
      verificationCode: "CODE-1234",
    });

    expect(screen.getByTestId("first-run-codex-code")).toHaveTextContent("CODE-1234");
    expect(screen.getByRole("link", { name: "https://auth.openai.com/codex/device" })).toHaveAttribute(
      "href",
      "https://auth.openai.com/codex/device",
    );
    expect(screen.getByText(copy.subscriptionDeviceSettings)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect ChatGPT" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Copy sign-in code" }));

    expect(writeText).toHaveBeenCalledWith("CODE-1234");
    expect(screen.getByRole("button", { name: "Sign-in code copied" })).toBeInTheDocument();
  });

  it("waits for a device code without saying sign-in is finishing", async () => {
    const onCancel = vi.fn();
    renderPanel({ status: "pending", reservationHeld: true }, { onCancel });

    expect(screen.getByRole("status")).toHaveTextContent("Wait for the ChatGPT sign-in code.");
    expect(screen.queryByText(copy.subscriptionFinishing)).not.toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: copy.subscriptionCancel }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("waits for cleanup without offering to cancel a saved sign-in", () => {
    renderPanel({ status: "finishing", reservationHeld: true });

    expect(screen.getByText(copy.subscriptionFinishing)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: copy.subscriptionCancel })).not.toBeInTheDocument();
  });

  it("shows cancellation progress and hides the device code until cleanup finishes", () => {
    renderPanel({
      status: "canceling",
      reservationHeld: true,
      verificationUrl: "https://auth.openai.com/codex/device",
      verificationCode: "CODE-1234",
    });

    expect(screen.getByRole("status")).toHaveTextContent("Canceling ChatGPT sign-in. Wait for cleanup to finish.");
    expect(screen.queryByText("CODE-1234")).not.toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: copy.subscriptionCancel })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect ChatGPT" })).not.toBeInTheDocument();
  });

  it("names the connected ChatGPT account", () => {
    renderPanel({ status: "ready", accountLabel: "ada@example.com", reservationHeld: false });

    expect(screen.getByRole("status")).toHaveTextContent("ChatGPT is connected as ada@example.com.");
    expect(screen.queryByRole("button", { name: "Connect ChatGPT" })).not.toBeInTheDocument();
  });

  it("explains delayed cleanup and offers another model source", () => {
    vi.useFakeTimers();
    try {
      renderPanel({ status: "finishing", reservationHeld: true });
      act(() => vi.advanceTimersByTime(60000));

      expect(screen.getByRole("status")).toHaveTextContent(
        "Connection setup is delayed. SuperPlane will keep checking. You can choose another model source.",
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it("does not claim a lost sign-in was saved while cleanup is pending", () => {
    renderPanel({ status: "reconnect_required", reservationHeld: true });

    expect(screen.getByRole("status")).toHaveTextContent("Wait for the previous connection attempt to stop.");
    expect(screen.queryByText(copy.subscriptionFinishing)).not.toBeInTheDocument();
  });
});
