import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { SupportSessionReadOnly } from "@/components/SupportSessionReadOnly";
import { SUPPORT_SESSION_READ_ONLY_MESSAGE } from "@/lib/supportSession";

const accountState = { supportSession: false };

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: { id: "acc-1", support_session: accountState.supportSession ? { active: true } : undefined },
  }),
}));

function renderControls() {
  render(
    <SupportSessionReadOnly>
      <input aria-label="Name" />
      <button type="button">Save</button>
    </SupportSessionReadOnly>,
  );
}

describe("SupportSessionReadOnly", () => {
  beforeEach(() => {
    accountState.supportSession = false;
  });

  it("leaves controls enabled outside a support session", () => {
    renderControls();

    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    expect(screen.getByRole("textbox", { name: "Name" })).toBeEnabled();
    expect(screen.queryByText(SUPPORT_SESSION_READ_ONLY_MESSAGE)).not.toBeInTheDocument();
  });

  it("disables controls and explains why in a support session", () => {
    accountState.supportSession = true;

    renderControls();

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("textbox", { name: "Name" })).toBeDisabled();
    expect(screen.getByText(SUPPORT_SESSION_READ_ONLY_MESSAGE)).toBeInTheDocument();
  });
});
