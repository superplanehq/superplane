import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { ACCOUNT_REDESIGN_NOTIFICATIONS, ACCOUNT_REDESIGN_SECURE_PROFILE } from "./accountProfileRedesignMocks";
import { AccountNotificationsRedesignPage } from "./AccountNotificationsRedesignPage";
import { AccountProfileRedesignPlayground } from "./AccountProfileRedesignPlayground";

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPlayground(page: "profile" | "security" = "profile") {
  return render(
    <ThemeProvider>
      <TooltipProvider>
        <AccountProfileRedesignPlayground
          initialPage={page}
          initialProfile={page === "security" ? ACCOUNT_REDESIGN_SECURE_PROFILE : undefined}
        />
      </TooltipProvider>
    </ThemeProvider>,
  );
}

describe("AccountProfileRedesignPlayground", () => {
  beforeEach(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => undefined;
    Element.prototype.releasePointerCapture ??= () => undefined;
    Element.prototype.scrollIntoView ??= () => undefined;
  });

  it("shows Account copy and the redesigned Account nav", () => {
    renderPlayground();

    expect(screen.getByTestId("workspace-page-header-title")).toHaveTextContent("Account");
    expect(
      screen.getByText("Preferences, profile information, and security for your SuperPlane account."),
    ).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-velocity-github")).toHaveTextContent("GitHub for Velocity");
    expect(screen.getByRole("button", { name: "Link GitHub" })).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-nav-account-profile")).toHaveTextContent("Account");
    expect(screen.queryByTestId("account-redesign-nav-account-security")).not.toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-nav-account-notifications")).toHaveTextContent("Notifications");
    expect(screen.queryByTestId("account-redesign-nav-account-preferences")).not.toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-nav-account-general")).not.toBeInTheDocument();
    expect(screen.queryByText("Leave workspace")).not.toBeInTheDocument();
    expect(screen.queryByText("Delete account")).not.toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-danger")).not.toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-appearance")).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-theme")).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-security")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Security & access" })).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-user-id")).not.toBeInTheDocument();
    expect(screen.queryByText(/User ID/)).not.toBeInTheDocument();
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

  it("links GitHub for Velocity from Profile", async () => {
    const user = userEvent.setup();
    renderPlayground();

    await user.click(screen.getByRole("button", { name: "Link GitHub" }));
    expect(screen.getByTestId("account-redesign-velocity-github")).toHaveTextContent("Linked as ada");
  });

  it("lets the user switch the profile email across sign-in methods", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TooltipProvider>
          <AccountProfileRedesignPlayground initialPage="profile" initialProfile={ACCOUNT_REDESIGN_SECURE_PROFILE} />
        </TooltipProvider>
      </ThemeProvider>,
    );

    await user.click(screen.getByTestId("account-redesign-email"));
    await user.click(await screen.findByRole("option", { name: /ada@users.noreply.github.com/ }));
    expect(screen.getByTestId("account-redesign-identity")).toHaveTextContent("ada@users.noreply.github.com");
  });

  it("moves the profile email to the remaining sign-in method after disconnect", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TooltipProvider>
          <AccountProfileRedesignPlayground initialPage="security" initialProfile={ACCOUNT_REDESIGN_SECURE_PROFILE} />
        </TooltipProvider>
      </ThemeProvider>,
    );

    await user.click(
      within(screen.getByTestId("account-redesign-sso-google")).getByRole("button", { name: "Disconnect" }),
    );
    await user.click(screen.getByRole("button", { name: "Disconnect Google" }));

    await user.click(screen.getByTestId("account-redesign-nav-account-profile"));
    expect(screen.getByTestId("account-redesign-identity")).toHaveTextContent("ada@users.noreply.github.com");
    expect(screen.getByTestId("account-redesign-email")).toHaveTextContent("ada@users.noreply.github.com");
  });

  it("hides the password row when the account has no password", () => {
    render(
      <ThemeProvider>
        <TooltipProvider>
          <AccountProfileRedesignPlayground
            initialPage="security"
            initialProfile={{ ...ACCOUNT_REDESIGN_SECURE_PROFILE, passwordSet: false }}
          />
        </TooltipProvider>
      </ThemeProvider>,
    );

    expect(screen.queryByTestId("account-redesign-password")).not.toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-signin")).toBeInTheDocument();
  });

  it("disables last SSO disconnect when no password is set", () => {
    render(
      <ThemeProvider>
        <TooltipProvider>
          <AccountProfileRedesignPlayground
            initialPage="security"
            initialProfile={{
              ...ACCOUNT_REDESIGN_SECURE_PROFILE,
              passwordSet: false,
              ssoAccounts: [
                { provider: "github", identity: "ada" },
                { provider: "google", identity: null },
              ],
            }}
          />
        </TooltipProvider>
      </ThemeProvider>,
    );

    expect(
      within(screen.getByTestId("account-redesign-sso-github")).getByRole("button", { name: "Disconnect" }),
    ).toBeDisabled();
    expect(screen.getByText("Keep at least one sign-in method.")).toBeInTheDocument();
  });

  it("shows Security & access on the Account page with password, SSO methods, and tokens", () => {
    renderPlayground();

    expect(screen.getByRole("heading", { name: "Security & access" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Sign in methods" })).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-password")).toHaveTextContent("Password is set.");
    expect(screen.getByTestId("account-redesign-sso-github")).toHaveTextContent("Connected as ada");
    expect(screen.getByTestId("account-redesign-sso-google")).toHaveTextContent("Not connected");
    expect(screen.getByRole("button", { name: "Sign in with Google" })).toBeInTheDocument();
    expect(
      screen.getByText("This token acts as you. Organization API keys act as the organization."),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-sessions")).not.toBeInTheDocument();
    expect(screen.queryByText("Two-factor authentication")).not.toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-velocity-github")).toHaveTextContent("GitHub for Velocity");
  });

  it("connects Google and disconnects GitHub on the same account", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TooltipProvider>
          <AccountProfileRedesignPlayground initialPage="security" />
        </TooltipProvider>
      </ThemeProvider>,
    );

    expect(screen.getByTestId("account-redesign-sso-github")).toHaveTextContent("Connected as ada");
    expect(screen.getByTestId("account-redesign-sso-google")).toHaveTextContent("Not connected");

    await user.click(screen.getByRole("button", { name: "Sign in with Google" }));
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
    expect(screen.getByTestId("account-redesign-nav-account-profile")).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-nav-account-notifications")).not.toBeInTheDocument();
  });
});
