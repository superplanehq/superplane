import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

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
  it("shows the device link and code while sign-in is pending", () => {
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
  });

  it("waits for a device code without saying sign-in is finishing", async () => {
    const onCancel = vi.fn();
    renderPanel({ status: "pending", reservationHeld: true }, { onCancel });

    expect(screen.getByRole("status")).toHaveTextContent("Wait for the ChatGPT sign-in code.");
    expect(screen.queryByText(copy.subscriptionFinishing)).not.toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: copy.subscriptionCancel }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("lets the user cancel while the login runner is finishing", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    renderPanel({ status: "finishing", reservationHeld: true }, { onCancel });

    expect(screen.getByText(copy.subscriptionFinishing)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: copy.subscriptionCancel }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("names the connected ChatGPT account", () => {
    renderPanel({ status: "ready", accountLabel: "ada@example.com", reservationHeld: false });

    expect(screen.getByRole("status")).toHaveTextContent("ChatGPT is connected as ada@example.com.");
    expect(screen.queryByRole("button", { name: "Connect ChatGPT" })).not.toBeInTheDocument();
  });
});
