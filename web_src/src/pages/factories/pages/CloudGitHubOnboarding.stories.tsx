import {
  GITHUB_INSTALL_REQUEST_NEXT,
  githubInstallRequestBody,
  githubInstallRequestSettingsTitle,
} from "@/lib/githubInstallRequestCopy";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";
import { CircleX } from "lucide-react";

import { GitHubInstallApprovedPage } from "@/pages/github/GitHubInstallApprovedPage";
import { HostedGitHubInstallPicker } from "@/pages/organization/settings/components/HostedGitHubInstallPicker";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";

import { FIRST_RUN_COPY } from "./onboarding/first-run/firstRunCopy";
import {
  CLOUD_GITHUB_ACME,
  CLOUD_GITHUB_APP_SLUG,
  CLOUD_GITHUB_LOGIN,
  CLOUD_GITHUB_OCTO,
  CLOUD_GITHUB_STATE,
  FIRST_RUN_REPOSITORIES,
  firstRunStoryChrome,
} from "./onboarding/first-run/firstRunMocks";
import { FirstRunChooseScreen } from "./onboarding/first-run/FirstRunChooseScreen";
import {
  CloudGitHubConnectStory,
  CloudGitHubPathMap,
  CloudGitHubSettingsFrame,
} from "./onboarding/first-run/cloudGitHubOnboardingStories";

/**
 * Every SuperPlane screen on the cloud public GitHub App path. First-run
 * never offers a private App. Local setup without the hosted App uses a
 * different wizard and is not in this set.
 */
const meta = {
  title: "Factories/Pages/Cloud GitHub App",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const PathMap: Story = {
  name: "0 Map",
  render: () => <CloudGitHubPathMap />,
};

export const Connect: Story = {
  name: "1 Connect GitHub",
  render: () => <CloudGitHubConnectStory />,
};

export const ConnectError: Story = {
  name: "1 Connect GitHub (error)",
  render: () => <CloudGitHubConnectStory connectError={FIRST_RUN_COPY.connect.connectError} />,
};

export const ConnectLoading: Story = {
  name: "1 Connect GitHub (loading)",
  render: () => <CloudGitHubConnectStory loading />,
};

export const AccountPicker: Story = {
  name: "2 Select account",
  render: () => (
    <CloudGitHubConnectStory
      githubLogin={CLOUD_GITHUB_LOGIN}
      pendingInstallations={[CLOUD_GITHUB_ACME, CLOUD_GITHUB_OCTO]}
    />
  ),
};

export const PickerStillWaiting: Story = {
  name: "2 Select account (waiting)",
  render: () => (
    <CloudGitHubConnectStory
      installRequested
      githubOrganization="acme"
      githubLogin={CLOUD_GITHUB_LOGIN}
      pendingInstallations={[CLOUD_GITHUB_OCTO]}
    />
  ),
};

export const WaitingNamedOrg: Story = {
  name: "2 Select account (waiting only)",
  render: () => (
    <CloudGitHubConnectStory installRequested githubOrganization="acme" githubLogin={CLOUD_GITHUB_LOGIN} />
  ),
};

export const WaitingUnknownOrg: Story = {
  name: "2 Select account (waiting, unnamed)",
  render: () => <CloudGitHubConnectStory installRequested githubLogin={CLOUD_GITHUB_LOGIN} />,
};

export const PickerAfterApproval: Story = {
  name: "2 Select account (approved)",
  render: () => (
    <CloudGitHubConnectStory
      installRequested
      githubOrganization="acme"
      githubLogin={CLOUD_GITHUB_LOGIN}
      pendingInstallations={[CLOUD_GITHUB_ACME, CLOUD_GITHUB_OCTO]}
    />
  ),
};

export const PickerBinding: Story = {
  name: "2 Select account (binding)",
  render: () => (
    <CloudGitHubConnectStory
      githubLogin={CLOUD_GITHUB_LOGIN}
      pendingInstallations={[CLOUD_GITHUB_ACME, CLOUD_GITHUB_OCTO]}
      bindingInstallationId={CLOUD_GITHUB_ACME.id}
    />
  ),
};

export const ChooseRepository: Story = {
  name: "3 Choose repository",
  render: () => (
    <FirstRunChooseScreen
      repositories={FIRST_RUN_REPOSITORIES}
      selectedRepository={null}
      chrome={firstRunStoryChrome(2)}
      onSelectRepository={(repository) => console.log("select repository", repository)}
      onEditConnection={() => console.log("edit connection")}
      onContinue={() => console.log("continue")}
    />
  ),
};

export const ChooseWhyMissing: Story = {
  name: "3 Choose repository (missing)",
  render: () => (
    <FirstRunChooseScreen
      repositories={FIRST_RUN_REPOSITORIES}
      selectedRepository="acme/api"
      chrome={firstRunStoryChrome(2)}
      initialWhyMissingOpen
      onSelectRepository={(repository) => console.log("select repository", repository)}
      onEditConnection={() => console.log("edit connection")}
      onContinue={() => console.log("continue")}
    />
  ),
};

export const ChooseLoading: Story = {
  name: "3 Choose repository (loading)",
  render: () => (
    <FirstRunChooseScreen
      repositories={FIRST_RUN_REPOSITORIES}
      selectedRepository={null}
      loading
      chrome={firstRunStoryChrome(2)}
      onSelectRepository={(repository) => console.log("select repository", repository)}
      onEditConnection={() => console.log("edit connection")}
      onContinue={() => console.log("continue")}
    />
  ),
};

export const AdminApproved: Story = {
  name: "Admin: request approved",
  render: () => (
    <MemoryRouter>
      <GitHubInstallApprovedPage />
    </MemoryRouter>
  ),
};

export const SettingsWaiting: Story = {
  name: "Settings: waiting",
  render: () => (
    <CloudGitHubSettingsFrame>
      <Alert data-testid="github-install-requested">
        <AlertTitle>{githubInstallRequestSettingsTitle()}</AlertTitle>
        <AlertDescription>
          <p>{githubInstallRequestBody()}</p>
          <p>{GITHUB_INSTALL_REQUEST_NEXT}</p>
        </AlertDescription>
      </Alert>
    </CloudGitHubSettingsFrame>
  ),
};

export const SettingsWaitingNamedOrg: Story = {
  name: "Settings: waiting (named org)",
  render: () => (
    <CloudGitHubSettingsFrame>
      <Alert data-testid="github-install-requested">
        <AlertTitle>{githubInstallRequestSettingsTitle("acme")}</AlertTitle>
        <AlertDescription>
          <p>{githubInstallRequestBody("acme")}</p>
          <p>{GITHUB_INSTALL_REQUEST_NEXT}</p>
        </AlertDescription>
      </Alert>
    </CloudGitHubSettingsFrame>
  ),
};

export const SettingsPicker: Story = {
  name: "Settings: account list",
  render: () => (
    <CloudGitHubSettingsFrame>
      <HostedGitHubInstallPicker
        installations={[CLOUD_GITHUB_ACME, CLOUD_GITHUB_OCTO]}
        state={CLOUD_GITHUB_STATE}
        appSlug={CLOUD_GITHUB_APP_SLUG}
      />
    </CloudGitHubSettingsFrame>
  ),
};

export const SettingsConnectionIssue: Story = {
  name: "Settings: connection issue",
  render: () => (
    <CloudGitHubSettingsFrame>
      <Alert className="border-destructive/40 bg-destructive/10 text-destructive [&>svg+div]:translate-y-0 [&>svg]:top-[14px] [&>svg]:text-destructive">
        <CircleX className="size-4" />
        <AlertTitle>Connection issue</AlertTitle>
        <AlertDescription>installation is not allowed</AlertDescription>
      </Alert>
    </CloudGitHubSettingsFrame>
  ),
};
