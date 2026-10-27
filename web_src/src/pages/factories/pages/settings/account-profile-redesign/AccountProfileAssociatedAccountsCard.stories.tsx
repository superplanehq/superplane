import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  AccountProfileAssociatedAccountsCard,
  type GitHubPrAuthorizationState,
} from "./AccountProfileAssociatedAccountsCard";

const linkedGithub = [{ providerId: "583231", username: "ada" }];

function AssociatedAccountsPreview({ initial }: { initial: GitHubPrAuthorizationState }) {
  const [authorization, setAuthorization] = useState(initial);

  return (
    <div className="max-w-2xl p-6">
      <AccountProfileAssociatedAccountsCard
        githubAccounts={linkedGithub}
        githubPrAuthorization={authorization}
        onLinkGithub={() => undefined}
        onRemoveGithub={() => undefined}
        onGithubPrAuthorizationChange={(enabled) =>
          setAuthorization((current) => ({
            ...current,
            enabled,
            login: enabled ? "ada" : current.login,
            needsReauthorization: false,
          }))
        }
      />
    </div>
  );
}

/**
 * The "Create pull requests with my GitHub account" setting. Turning it on
 * sends the member to GitHub. The stories flip the switch locally.
 */
const meta = {
  title: "Factories/Pages/Settings/Account/Associated accounts",
  component: AssociatedAccountsPreview,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof AssociatedAccountsPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const PullRequestsAsMemberUnavailable: Story = {
  name: "Pull requests as member: unavailable",
  args: { initial: { available: false, enabled: false, needsReauthorization: false } },
};

export const PullRequestsAsMemberOff: Story = {
  name: "Pull requests as member: off",
  args: { initial: { available: true, enabled: false, needsReauthorization: false } },
};

export const PullRequestsAsMemberOn: Story = {
  name: "Pull requests as member: on",
  args: { initial: { available: true, enabled: true, login: "ada", needsReauthorization: false } },
};

export const PullRequestsAsMemberNeedsAuthorization: Story = {
  name: "Pull requests as member: needs authorization",
  args: { initial: { available: true, enabled: false, login: "ada", needsReauthorization: true } },
};
