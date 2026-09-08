import { lazy, Suspense, useEffect, useState, type ReactNode } from "react";

import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./FirstRunConnectScreen";
import {
  CLOUD_GITHUB_ACME,
  CLOUD_GITHUB_APP_SLUG,
  CLOUD_GITHUB_GLOBEX,
  CLOUD_GITHUB_LOGIN,
  CLOUD_GITHUB_OCTO,
  CLOUD_GITHUB_STATE,
  FIRST_RUN_REPOSITORIES,
  FIRST_RUN_STORY_EMAIL,
} from "./firstRunMocks";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";
import type { FirstRunAnalysisStatus, FirstRunChrome, FirstRunScreenId, FirstRunTicketSource } from "./firstRunTypes";
import { FirstRunWelcomeScreen } from "./FirstRunWelcomeScreen";

const FirstRunBoardExit = lazy(async () => {
  const module = await import("./FirstRunBoardExit");
  return { default: module.FirstRunBoardExit };
});

const STAGE_MS = 900;
const COMPLETE_AFTER_MS = STAGE_MS * 3;

/**
 * Clickable Storybook journey for the first-run PRD. Local state only.
 * Production wiring stays out of this file.
 */
export function FirstRunFlow({
  firstName,
  email = FIRST_RUN_STORY_EMAIL,
  initialScreen = "welcome",
  analysisStatus = "running",
  completeAfterMs = COMPLETE_AFTER_MS,
  board,
  onLogOut,
}: {
  firstName?: string;
  email?: string;
  initialScreen?: FirstRunScreenId;
  analysisStatus?: FirstRunAnalysisStatus;
  completeAfterMs?: number;
  board?: ReactNode;
  onLogOut?: () => void;
}) {
  const [screen, setScreen] = useState<FirstRunScreenId>(initialScreen);
  const [pickerShowing, setPickerShowing] = useState(false);
  const [ticketSource, setTicketSource] = useState<FirstRunTicketSource | null>(null);
  const [selectedRepository, setSelectedRepository] = useState<string | null>(null);
  const [stageIndex, setStageIndex] = useState(0);

  const openAccountPicker = () => {
    setPickerShowing(true);
    setScreen("connect");
  };

  const chromeFor = (stepIndex: number, onBack?: () => void): FirstRunChrome => ({
    displayName: firstName,
    email,
    onLogOut,
    stepIndex,
    onBack,
  });

  useEffect(() => {
    if (screen !== "analysis" || analysisStatus === "failed") return;

    const stageTimer = window.setInterval(() => {
      setStageIndex((current) => Math.min(current + 1, 2));
    }, STAGE_MS);
    const doneTimer =
      analysisStatus === "running" ? window.setTimeout(() => setScreen("board"), completeAfterMs) : undefined;

    return () => {
      window.clearInterval(stageTimer);
      if (doneTimer) window.clearTimeout(doneTimer);
    };
  }, [analysisStatus, completeAfterMs, screen]);

  if (screen === "welcome") {
    return (
      <FirstRunWelcomeScreen firstName={firstName} chrome={chromeFor(0)} onGetStarted={() => setScreen("connect")} />
    );
  }

  if (screen === "connect") {
    return (
      <FirstRunConnectScreen
        chrome={chromeFor(1, pickerShowing ? () => setPickerShowing(false) : () => setScreen("welcome"))}
        githubAppSlug={CLOUD_GITHUB_APP_SLUG}
        githubState={CLOUD_GITHUB_STATE}
        githubLogin={pickerShowing ? CLOUD_GITHUB_LOGIN : ""}
        installRequested={pickerShowing}
        githubOrganization={pickerShowing ? CLOUD_GITHUB_GLOBEX : ""}
        pendingInstallations={pickerShowing ? [CLOUD_GITHUB_ACME, CLOUD_GITHUB_OCTO] : []}
        onConnectGitHub={() => setPickerShowing(true)}
        onUseInstallation={() => setScreen("choose")}
      />
    );
  }

  if (screen === "choose") {
    return (
      <FirstRunChooseScreen
        repositories={FIRST_RUN_REPOSITORIES}
        selectedRepository={selectedRepository}
        chrome={chromeFor(2, openAccountPicker)}
        onSelectRepository={setSelectedRepository}
        onEditConnection={openAccountPicker}
        onContinue={() => {
          if (selectedRepository) setScreen("tickets");
        }}
      />
    );
  }

  if (screen === "tickets") {
    return (
      <FirstRunTicketsScreen
        ticketSource={ticketSource}
        chrome={chromeFor(3, () => setScreen("choose"))}
        onSelectTicketSource={setTicketSource}
        onAnalyzeTickets={() => {
          if (!ticketSource) return;
          setStageIndex(0);
          setScreen("analysis");
        }}
      />
    );
  }

  if (screen === "analysis") {
    return (
      <FirstRunAnalysisScreen
        status={analysisStatus}
        currentStageIndex={stageIndex}
        chrome={chromeFor(4)}
        onRetry={() => {
          setStageIndex(0);
          setScreen("analysis");
        }}
      />
    );
  }

  if (board) {
    return board;
  }

  return (
    <Suspense fallback={<div data-testid="first-run-board" />}>
      <FirstRunBoardExit />
    </Suspense>
  );
}
