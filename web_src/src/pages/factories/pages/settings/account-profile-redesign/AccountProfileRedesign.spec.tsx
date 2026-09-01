import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ACCOUNT_REDESIGN_SECURE_PROFILE } from "./accountProfileRedesignMocks";
import { AccountProfileRedesignPlayground } from "./AccountProfileRedesignPlayground";

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPlayground(page: "profile" | "security" | "preferences" = "profile") {
  return render(
    <AccountProfileRedesignPlayground
      initialPage={page}
      initialProfile={page === "security" ? ACCOUNT_REDESIGN_SECURE_PROFILE : undefined}
    />,
  );
}

function mockClipboard() {
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
  return writeText;
}

describe("AccountProfileRedesignPlayground", () => {
  beforeEach(() => {
    mockClipboard();
  });

  it("shows Profile copy and the redesigned Account nav", () => {
    renderPlayground();

    expect(screen.getByRole("heading", { name: "Profile" })).toBeInTheDocument();
    expect(screen.getByText("Manage how your name and email appear in SuperPlane.")).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-nav-account-profile")).toHaveTextContent("Profile");
    expect(screen.getByTestId("account-redesign-nav-account-security")).toHaveTextContent("Security");
    expect(screen.getByTestId("account-redesign-nav-account-preferences")).toHaveTextContent("Preferences");
    expect(screen.queryByTestId("account-redesign-nav-account-general")).not.toBeInTheDocument();
  });

  it("keeps Save disabled until the name changes, then saves", async () => {
    const user = userEvent.setup();
    renderPlayground();

    const save = screen.getByTestId("account-redesign-save");
    expect(save).toBeDisabled();

    await user.clear(screen.getByTestId("account-redesign-name"));
    await user.type(screen.getByTestId("account-redesign-name"), "Ada Byron");
    expect(save).toBeEnabled();

    await user.click(save);
    expect(save).toBeDisabled();
  });

  it("copies the user ID", async () => {
    const user = userEvent.setup();
    const writeText = mockClipboard();
    renderPlayground();

    await user.click(screen.getByRole("button", { name: "Copy" }));
    expect(writeText).toHaveBeenCalledWith("5f76536d-bc02-4f99-81e6-e159ac40ebbb");
    expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it("opens Security with password, tokens, and sessions", async () => {
    const user = userEvent.setup();
    renderPlayground();

    await user.click(screen.getByTestId("account-redesign-nav-account-security"));
    expect(screen.getByRole("heading", { name: "Security" })).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-password")).toHaveTextContent("Password is set.");
    expect(
      screen.getByText("This token acts as you. Organization API keys act as the organization."),
    ).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-sessions")).toBeInTheDocument();
  });

  it("filters the settings nav", async () => {
    const user = userEvent.setup();
    renderPlayground();

    await user.type(screen.getByTestId("account-redesign-find"), "sec");
    expect(screen.getByTestId("account-redesign-nav-account-security")).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-nav-account-profile")).not.toBeInTheDocument();
  });

  it("shows theme and timezone on Preferences", () => {
    renderPlayground("preferences");

    expect(screen.getByRole("heading", { name: "Preferences" })).toBeInTheDocument();
    expect(screen.getByRole("radiogroup", { name: "Theme" })).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-timezone")).toBeInTheDocument();
  });
});
