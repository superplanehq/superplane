import type { Meta, StoryObj } from "@storybook/react-vite";

import { FIRST_RUN_COPY } from "./onboarding/first-run/firstRunCopy";
import { firstRunStoryChrome } from "./onboarding/first-run/firstRunMocks";
import { FirstRunAnalysisScreen } from "./onboarding/first-run/FirstRunAnalysisScreen";
import { FirstRunBoardExit } from "./onboarding/first-run/FirstRunBoardExit";
import { FirstRunConnectScreen } from "./onboarding/first-run/FirstRunConnectScreen";
import { FirstRunFlow } from "./onboarding/first-run/FirstRunFlow";

/**
 * Isolated first-run screens. Step stories mount the clickable flow so
 * primary buttons advance.
 */
const meta = {
  title: "Factories/Pages/First run",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Journey: Story = {
  name: "0 Clickable journey",
  render: () => <FirstRunFlow firstName="Ada" />,
};

export const Welcome: Story = {
  name: "1 Welcome",
  render: () => <FirstRunFlow firstName="Ada" />,
};

export const Connect: Story = {
  name: "2a Connect GitHub",
  render: () => <FirstRunFlow firstName="Ada" initialScreen="connect" />,
};

export const ConnectGitHubConnected: Story = {
  name: "2b Connect GitHub (connected)",
  render: () => (
    <FirstRunConnectScreen
      githubConnected
      chrome={firstRunStoryChrome(1)}
      onConnectGitHub={() => undefined}
      onContinue={() => undefined}
    />
  ),
};

export const ConnectError: Story = {
  name: "2c Connect GitHub (error)",
  render: () => (
    <FirstRunConnectScreen
      githubConnected={false}
      connectError={FIRST_RUN_COPY.connect.connectError}
      chrome={firstRunStoryChrome(1)}
      onConnectGitHub={() => undefined}
      onContinue={() => undefined}
    />
  ),
};

export const ConnectInstallRequested: Story = {
  name: "2d Connect GitHub (waiting for approval)",
  render: () => (
    <FirstRunConnectScreen
      githubConnected={false}
      installRequested
      githubOrganization="acme"
      chrome={firstRunStoryChrome(1)}
      onConnectGitHub={() => undefined}
      onContinue={() => undefined}
    />
  ),
};

export const Choose: Story = {
  name: "3 Choose repository",
  render: () => <FirstRunFlow firstName="Ada" initialScreen="choose" />,
};

export const Tickets: Story = {
  name: "4 Connect ticket system",
  render: () => <FirstRunFlow firstName="Ada" initialScreen="tickets" />,
};

export const Analysis: Story = {
  name: "5a Analysis",
  render: () => <FirstRunFlow firstName="Ada" initialScreen="analysis" />,
};

export const AnalysisOverrun: Story = {
  name: "5b Analysis (overrun)",
  render: () => (
    <FirstRunAnalysisScreen
      status="overrun"
      currentStageIndex={2}
      chrome={firstRunStoryChrome(4)}
      onRetry={() => undefined}
    />
  ),
};

export const AnalysisFailed: Story = {
  name: "5c Analysis (failed)",
  render: () => (
    <FirstRunAnalysisScreen
      status="failed"
      currentStageIndex={0}
      chrome={firstRunStoryChrome(4)}
      onRetry={() => undefined}
    />
  ),
};

export const Board: Story = {
  name: "6 Board",
  render: () => <FirstRunBoardExit />,
};
