import { useQueryClient } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";

import { accountOrganizationsQueryKey } from "@/hooks/useAccountOrganizations";
import { FIRST_RUN_COPY } from "./onboarding/first-run/firstRunCopy";
import {
  FIRST_RUN_STORY_ORGANIZATION_ID,
  FIRST_RUN_STORY_ORGANIZATION_NAME,
  firstRunStoryChrome,
} from "./onboarding/first-run/firstRunMocks";
import { FirstRunAnalysisScreen } from "./onboarding/first-run/FirstRunAnalysisScreen";
import { FirstRunBoardExit } from "./onboarding/first-run/FirstRunBoardExit";
import { FirstRunConnectScreen } from "./onboarding/first-run/FirstRunConnectScreen";
import { FirstRunFlow } from "./onboarding/first-run/FirstRunFlow";

/**
 * Every screen's account menu needs a router (Profile, settings links) and,
 * if a reviewer opens "Switch organization", account data to list. `Board`
 * mounts `FactoriesHarness`, which brings its own router, so it renders
 * without this frame instead.
 */
function FirstRunStoryFrame({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  queryClient.setQueryData(accountOrganizationsQueryKey, [
    { id: FIRST_RUN_STORY_ORGANIZATION_ID, slug: "acme", name: FIRST_RUN_STORY_ORGANIZATION_NAME },
    { id: "org-storybook-other", slug: "other-co", name: "Other Co" },
  ]);
  return <MemoryRouter>{children}</MemoryRouter>;
}

/**
 * Isolated first-run screens. Step stories mount the clickable flow so
 * primary buttons advance. Every screen carries the bottom-left account menu;
 * the `(sign out only)` stories show the fallback with nowhere else to go.
 */
const meta = {
  title: "Factories/Pages/First run",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Journey: Story = {
  name: "0 Clickable journey",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" />
    </FirstRunStoryFrame>
  ),
};

export const Welcome: Story = {
  name: "1 Welcome",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" />
    </FirstRunStoryFrame>
  ),
};

export const WelcomeSignOutOnly: Story = {
  name: "1 Welcome (sign out only)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" canQuitOnboarding={false} />
    </FirstRunStoryFrame>
  ),
};

export const Connect: Story = {
  name: "2a Connect GitHub",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="connect" />
    </FirstRunStoryFrame>
  ),
};

export const ConnectSignOutOnly: Story = {
  name: "2a Connect GitHub (sign out only)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="connect" canQuitOnboarding={false} />
    </FirstRunStoryFrame>
  ),
};

export const ConnectGitHubConnected: Story = {
  name: "2b Connect GitHub (connected)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunConnectScreen
        githubConnected
        chrome={firstRunStoryChrome(1)}
        onConnectGitHub={() => console.log("connect GitHub")}
        onContinue={() => console.log("continue")}
      />
    </FirstRunStoryFrame>
  ),
};

export const ConnectError: Story = {
  name: "2c Connect GitHub (error)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunConnectScreen
        githubConnected={false}
        connectError={FIRST_RUN_COPY.connect.connectError}
        chrome={firstRunStoryChrome(1)}
        onConnectGitHub={() => console.log("connect GitHub")}
        onContinue={() => console.log("continue")}
      />
    </FirstRunStoryFrame>
  ),
};

export const ConnectInstallRequested: Story = {
  name: "2d Connect GitHub (waiting for approval)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunConnectScreen
        githubConnected={false}
        installRequested
        githubOrganization="acme"
        chrome={firstRunStoryChrome(1)}
        onConnectGitHub={() => console.log("connect GitHub")}
        onContinue={() => console.log("continue")}
      />
    </FirstRunStoryFrame>
  ),
};

export const Choose: Story = {
  name: "3 Choose repository",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="choose" />
    </FirstRunStoryFrame>
  ),
};

export const ChooseSignOutOnly: Story = {
  name: "3 Choose repository (sign out only)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="choose" canQuitOnboarding={false} />
    </FirstRunStoryFrame>
  ),
};

export const Tickets: Story = {
  name: "4 Connect ticket system",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="tickets" />
    </FirstRunStoryFrame>
  ),
};

export const TicketsSignOutOnly: Story = {
  name: "4 Connect ticket system (sign out only)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="tickets" canQuitOnboarding={false} />
    </FirstRunStoryFrame>
  ),
};

export const Analysis: Story = {
  name: "5a Analysis",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="analysis" />
    </FirstRunStoryFrame>
  ),
};

export const AnalysisSignOutOnly: Story = {
  name: "5a Analysis (sign out only)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunFlow firstName="Ada" initialScreen="analysis" canQuitOnboarding={false} />
    </FirstRunStoryFrame>
  ),
};

export const AnalysisOverrun: Story = {
  name: "5b Analysis (overrun)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunAnalysisScreen
        status="overrun"
        currentStageIndex={2}
        chrome={firstRunStoryChrome(4)}
        onRetry={() => console.log("retry")}
      />
    </FirstRunStoryFrame>
  ),
};

export const AnalysisFailed: Story = {
  name: "5c Analysis (failed)",
  render: () => (
    <FirstRunStoryFrame>
      <FirstRunAnalysisScreen
        status="failed"
        currentStageIndex={0}
        chrome={firstRunStoryChrome(4)}
        onRetry={() => console.log("retry")}
      />
    </FirstRunStoryFrame>
  ),
};

export const Board: Story = {
  name: "6 Board",
  render: () => <FirstRunBoardExit />,
};
