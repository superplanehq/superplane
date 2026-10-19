import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  bitbucketOnboarding,
  chooseBitbucket,
  renderSetup,
  resetBitbucketFirstRunHarness,
} from "./FirstRunSetup.bitbucket.spec.fixtures";

describe("FirstRunSetup Bitbucket account switcher", () => {
  const ada = { login: "ada-bb", providerUserId: "11111111-1111-1111-1111-111111111111" };
  const grace = { login: "grace-bb", providerUserId: "22222222-2222-2222-2222-222222222222" };

  beforeEach(() => {
    resetBitbucketFirstRunHarness();
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = ada;
    bitbucketOnboarding.repositories = [{ fullName: "acme-team/api" }];
  });

  it("lists linked accounts like GitHub and blocks the screen while a switch runs", async () => {
    bitbucketOnboarding.identities = [ada, grace];
    let finishSwitch: (() => void) | undefined;
    bitbucketOnboarding.selectIdentity = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finishSwitch = resolve;
        }),
    );
    const user = userEvent.setup();
    renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.getByTestId("first-run-bitbucket-signed-in-as")).toHaveTextContent(
      "You are signed in to Bitbucket as ada-bb.",
    );
    const switchButton = screen.getByTestId("first-run-switch-bitbucket-account");
    expect(switchButton).toHaveTextContent("Switch account");

    await user.click(switchButton);
    const menuEntries = Array.from(
      screen.getByRole("menu").querySelectorAll('[role="menuitemradio"], [role="separator"], [role="menuitem"]'),
    ).map((entry) => `${entry.getAttribute("role")}:${entry.textContent}`);
    expect(menuEntries).toEqual([
      "menuitemradio:ada-bb",
      "menuitemradio:grace-bb",
      "separator:",
      "menuitem:Connect another Bitbucket account",
    ]);
    expect(screen.getByRole("menuitemradio", { name: "ada-bb" })).toHaveAttribute("aria-checked", "true");

    await user.click(screen.getByRole("menuitemradio", { name: "grace-bb" }));

    expect(bitbucketOnboarding.selectIdentity).toHaveBeenCalledWith(grace.providerUserId);
    expect(screen.getByTestId("first-run-switch-bitbucket-account")).toBeDisabled();
    await act(async () => finishSwitch?.());
    expect(screen.getByTestId("first-run-switch-bitbucket-account")).toBeEnabled();
  });

  it("explains a failed account switch and clears the message on the next attempt", async () => {
    bitbucketOnboarding.identities = [ada, grace];
    let rejectSwitch: ((error: Error) => void) | undefined;
    let finishSwitch: (() => void) | undefined;
    bitbucketOnboarding.selectIdentity = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise<void>((_, reject) => {
            rejectSwitch = reject;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            finishSwitch = resolve;
          }),
      );
    const user = userEvent.setup();
    renderSetup(new Set());
    await chooseBitbucket(user);

    await user.click(screen.getByTestId("first-run-switch-bitbucket-account"));
    await user.click(screen.getByRole("menuitemradio", { name: "grace-bb" }));
    await act(async () => rejectSwitch?.(new Error("denied")));

    expect(await screen.findByTestId("first-run-bitbucket-switch-error")).toHaveTextContent(
      "SuperPlane could not switch the Bitbucket account. Try again.",
    );
    expect(screen.getByTestId("first-run-bitbucket-signed-in-as")).toHaveTextContent(
      "You are signed in to Bitbucket as ada-bb.",
    );

    await user.click(screen.getByTestId("first-run-switch-bitbucket-account"));
    await user.click(screen.getByRole("menuitemradio", { name: "grace-bb" }));

    expect(screen.queryByTestId("first-run-bitbucket-switch-error")).not.toBeInTheDocument();
    await act(async () => finishSwitch?.());
    expect(screen.queryByTestId("first-run-bitbucket-switch-error")).not.toBeInTheDocument();
  });

  it("offers to connect another account when only one account is linked", async () => {
    bitbucketOnboarding.identities = [ada];
    const previousAssign = window.location.assign.bind(window.location);
    const assign = vi.fn();
    window.location.assign = assign;
    const user = userEvent.setup();

    try {
      renderSetup(new Set());
      await chooseBitbucket(user);
      await user.click(screen.getByTestId("first-run-switch-bitbucket-account"));
      await user.click(screen.getByRole("menuitem", { name: "Connect another Bitbucket account" }));

      expect(assign).toHaveBeenCalledWith(expect.stringContaining("/auth/bitbucket?intent=connect"));
      expect(bitbucketOnboarding.selectIdentity).not.toHaveBeenCalled();
    } finally {
      window.location.assign = previousAssign;
    }
  });
});
