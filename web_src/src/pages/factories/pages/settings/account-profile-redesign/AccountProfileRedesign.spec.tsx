import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ACCOUNT_REDESIGN_NOTIFICATIONS, ACCOUNT_REDESIGN_SECURE_PROFILE } from "./accountProfileRedesignMocks";
import { AccountNotificationsRedesignPage } from "./AccountNotificationsRedesignPage";
import { AccountProfileRedesignPlayground } from "./AccountProfileRedesignPlayground";

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPlayground(page: "profile" | "security" = "profile") {
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
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => undefined;
    Element.prototype.releasePointerCapture ??= () => undefined;
    Element.prototype.scrollIntoView ??= () => undefined;
  });

  it("shows Profile copy and the redesigned Account nav", () => {
    renderPlayground();

    expect(screen.getByRole("heading", { name: "Profile" })).toBeInTheDocument();
    expect(screen.getByText("Manage how your name appears in SuperPlane.")).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-nav-account-profile")).toHaveTextContent("Profile");
    expect(screen.getByTestId("account-redesign-nav-account-linked-accounts")).toHaveTextContent("Linked accounts");
    expect(screen.getByTestId("account-redesign-nav-account-security")).toHaveTextContent("Security");
    expect(screen.queryByTestId("account-redesign-nav-account-notifications")).not.toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-nav-account-preferences")).not.toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-nav-account-general")).not.toBeInTheDocument();
    expect(screen.queryByText("Leave workspace")).not.toBeInTheDocument();
    expect(screen.queryByText("Delete account")).not.toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-danger")).not.toBeInTheDocument();
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

    expect(screen.getByTestId("account-redesign-user-id")).toHaveTextContent("5f76536d…ebbb");
    expect(screen.getByTestId("account-redesign-identity")).not.toHaveTextContent("User ID");

    await user.click(screen.getByRole("button", { name: "Copy user ID" }));
    expect(writeText).toHaveBeenCalledWith("5f76536d-bc02-4f99-81e6-e159ac40ebbb");
    expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it("lets the user switch the profile email across sign-in methods", async () => {
    const user = userEvent.setup();
    render(<AccountProfileRedesignPlayground initialPage="profile" initialProfile={ACCOUNT_REDESIGN_SECURE_PROFILE} />);

    await user.click(screen.getByTestId("account-redesign-email"));
    await user.click(await screen.findByRole("option", { name: /ada@users.noreply.github.com/ }));
    expect(screen.getByTestId("account-redesign-identity")).toHaveTextContent("ada@users.noreply.github.com");
  });

  it("moves the profile email to the remaining sign-in method after disconnect", async () => {
    const user = userEvent.setup();
    render(
      <AccountProfileRedesignPlayground
        initialPage="linked-accounts"
        initialProfile={ACCOUNT_REDESIGN_SECURE_PROFILE}
      />,
    );

    await user.click(
      within(screen.getByTestId("account-redesign-sso-google")).getByRole("button", { name: "Disconnect" }),
    );
    await user.click(screen.getByRole("button", { name: "Disconnect Google" }));

    await user.click(screen.getByTestId("account-redesign-nav-account-profile"));
    expect(screen.getByTestId("account-redesign-identity")).toHaveTextContent("ada@users.noreply.github.com");
    expect(screen.getByTestId("account-redesign-email")).toHaveValue("ada@users.noreply.github.com");
  });

  it("hides the password card when the account has no password", () => {
    render(
      <AccountProfileRedesignPlayground
        initialPage="security"
        initialProfile={{ ...ACCOUNT_REDESIGN_SECURE_PROFILE, passwordSet: false }}
      />,
    );

    expect(screen.queryByTestId("account-redesign-password-card")).not.toBeInTheDocument();
  });

  it("disables last SSO disconnect when no password is set", () => {
    render(
      <AccountProfileRedesignPlayground
        initialPage="linked-accounts"
        initialProfile={{
          ...ACCOUNT_REDESIGN_SECURE_PROFILE,
          passwordSet: false,
          ssoAccounts: [
            { provider: "github", identity: "ada" },
            { provider: "google", identity: null },
          ],
        }}
      />,
    );

    expect(
      within(screen.getByTestId("account-redesign-sso-github")).getByRole("button", { name: "Disconnect" }),
    ).toBeDisabled();
    expect(screen.getByText("Keep at least one sign-in method.")).toBeInTheDocument();
  });

  it("opens Security with password and tokens but no linked accounts", async () => {
    const user = userEvent.setup();
    renderPlayground();

    await user.click(screen.getByTestId("account-redesign-nav-account-security"));
    expect(screen.getByRole("heading", { name: "Security" })).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-password")).toHaveTextContent("Password is set.");
    expect(
      screen.getByText("This token acts as you. Organization API keys act as the organization."),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-sso")).not.toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-sessions")).not.toBeInTheDocument();
    expect(screen.queryByText("Two-factor authentication")).not.toBeInTheDocument();
  });

  it("opens Linked accounts with each provider and its connection state", async () => {
    const user = userEvent.setup();
    renderPlayground();

    await user.click(screen.getByTestId("account-redesign-nav-account-linked-accounts"));
    expect(screen.getByRole("heading", { name: "Linked accounts" })).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-sso-github")).toHaveTextContent("Connected as ada");
    expect(screen.getByTestId("account-redesign-sso-google")).toHaveTextContent("Not connected");
    expect(screen.getByText(/match your repository activity to you in Velocity reports/)).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-password-card")).not.toBeInTheDocument();
  });

  it("connects Google and disconnects GitHub on the same account", async () => {
    const user = userEvent.setup();
    render(<AccountProfileRedesignPlayground initialPage="linked-accounts" />);

    expect(screen.getByTestId("account-redesign-sso-github")).toHaveTextContent("Connected as ada");
    expect(screen.getByTestId("account-redesign-sso-google")).toHaveTextContent("Not connected");

    await user.click(screen.getByRole("button", { name: "Connect Google" }));
    expect(screen.getByTestId("account-redesign-sso-google")).toHaveTextContent("Connected as ada@example.com");

    await user.click(
      within(screen.getByTestId("account-redesign-sso-github")).getByRole("button", { name: "Disconnect" }),
    );
    await user.click(screen.getByRole("button", { name: "Disconnect GitHub" }));

    expect(screen.getByTestId("account-redesign-sso-github")).toHaveTextContent("Not connected");
    expect(screen.getByTestId("account-redesign-sso-google")).toHaveTextContent("Connected as ada@example.com");
  });

  it("turns task emails off and hides events", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const notifications = { ...ACCOUNT_REDESIGN_NOTIFICATIONS };

    const { rerender } = render(
      <AccountNotificationsRedesignPage
        email="ada@example.com"
        workspaces={[{ id: "ws-1", name: "Semaphore" }]}
        notifications={notifications}
        onChange={onChange}
        onSave={() => undefined}
      />,
    );

    expect(screen.getByText("Choose which task emails SuperPlane sends you.")).toBeInTheDocument();
    expect(screen.getByText("Added as a task owner")).toBeInTheDocument();

    await user.click(screen.getByRole("switch", { name: "Send task emails" }));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ emailEnabled: false }));

    rerender(
      <AccountNotificationsRedesignPage
        email="ada@example.com"
        workspaces={[{ id: "ws-1", name: "Semaphore" }]}
        notifications={{ ...notifications, emailEnabled: false }}
        onChange={onChange}
        onSave={() => undefined}
      />,
    );

    expect(screen.getByTestId("account-redesign-notifications-off")).toHaveTextContent("Task emails are off.");
    expect(screen.queryByText("Added as a task owner")).not.toBeInTheDocument();
  });

  it("filters the settings nav", async () => {
    const user = userEvent.setup();
    renderPlayground();

    await user.type(screen.getByTestId("account-redesign-find"), "sec");
    expect(screen.getByTestId("account-redesign-nav-account-security")).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-nav-account-profile")).not.toBeInTheDocument();
  });
});
