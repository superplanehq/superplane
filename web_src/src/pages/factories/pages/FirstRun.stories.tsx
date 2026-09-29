import type { Meta, StoryObj } from "@storybook/react-vite";

import { FIRST_RUN_COPY } from "./onboarding/first-run/firstRunCopy";
import { firstRunStoryChrome } from "./onboarding/first-run/firstRunMocks";
import { analysisSphereFor } from "./onboarding/first-run/firstRunSphereFor";
import { FirstRunAnalysisScreen } from "./onboarding/first-run/FirstRunAnalysisScreen";
import { FirstRunBoardExit } from "./onboarding/first-run/FirstRunBoardExit";
import { FirstRunConnectScreen } from "./onboarding/first-run/FirstRunConnectScreen";
import { FirstRunFlow } from "./onboarding/first-run/FirstRunFlow";
import type { FirstRunAnalysisProgress } from "./onboarding/first-run/firstRunAnalysisProgress";

/** Static analysis screen in one exact state, for design review. */
function AnalysisState({ progress, failed = false }: { progress: FirstRunAnalysisProgress; failed?: boolean }) {
  return (
    <FirstRunAnalysisScreen
      progress={progress}
      sourceName="GitHub issues"
      failed={failed}
      chrome={firstRunStoryChrome(4)}
      sphere={analysisSphereFor("acme/payments-service", progress.total)}
      onGoToBoard={() => undefined}
    />
  );
}

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

export const ConnectError: Story = {
  name: "2b Connect GitHub (error)",
  render: () => (
    <FirstRunConnectScreen
      connectError={FIRST_RUN_COPY.connect.connectError}
      chrome={firstRunStoryChrome(1)}
      onConnectGitHub={() => undefined}
    />
  ),
};

export const ConnectInstallRequested: Story = {
  name: "2c Connect GitHub (waiting for approval)",
  render: () => (
    <FirstRunConnectScreen
      installRequested
      githubOrganization="acme"
      chrome={firstRunStoryChrome(1)}
      onConnectGitHub={() => undefined}
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
  name: "5a Analysis (live demo)",
  render: () => <FirstRunFlow firstName="Ada" initialScreen="analysis" />,
};

export const AnalysisImporting: Story = {
  name: "5b Analysis (importing)",
  render: () => <AnalysisState progress={{ total: 0, scored: 0, ready: 0, stageIndex: 0 }} />,
};

export const AnalysisScoring: Story = {
  name: "5c Analysis (scoring)",
  render: () => <AnalysisState progress={{ total: 10, scored: 3, ready: 2, stageIndex: 1 }} />,
};

export const AnalysisScored: Story = {
  name: "5d Analysis (scored)",
  render: () => <AnalysisState progress={{ total: 10, scored: 10, ready: 6, stageIndex: 2 }} />,
};

export const AnalysisEmpty: Story = {
  name: "5e Analysis (no tickets)",
  render: () => <AnalysisState progress={{ total: 0, scored: 0, ready: 0, stageIndex: 0, empty: true }} />,
};

export const AnalysisFailed: Story = {
  name: "5f Analysis (failed)",
  render: () => <AnalysisState progress={{ total: 0, scored: 0, ready: 0, stageIndex: 0 }} failed />,
};

export const Board: Story = {
  name: "6 Board",
  render: () => <FirstRunBoardExit />,
};
