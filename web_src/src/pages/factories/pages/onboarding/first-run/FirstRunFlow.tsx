import { useEffect, useState, type ReactNode } from "react";

import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";
import { FirstRunBoardExit } from "./FirstRunBoardExit";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./FirstRunConnectScreen";
import { FIRST_RUN_REPOSITORIES, FIRST_RUN_STORY_EMAIL } from "./firstRunMocks";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";
import type { FirstRunAnalysisStatus, FirstRunChrome, FirstRunScreenId, FirstRunTicketSource } from "./firstRunTypes";
import { FirstRunWelcomeScreen } from "./FirstRunWelcomeScreen";

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
  githubStartsConnected = false,
  analysisStatus = "running",
  completeAfterMs = COMPLETE_AFTER_MS,
  board,
  onLogOut,
}: {
  firstName?: string;
  email?: string;
  initialScreen?: FirstRunScreenId;
  githubStartsConnected?: boolean;
  analysisStatus?: FirstRunAnalysisStatus;
  completeAfterMs?: number;
  board?: ReactNode;
  onLogOut?: () => void;
}) {
  const [screen, setScreen] = useState<FirstRunScreenId>(initialScreen);
  const [githubConnected, setGithubConnected] = useState(githubStartsConnected);
  const [ticketSource, setTicketSource] = useState<FirstRunTicketSource | null>(null);
  const [selectedRepository, setSelectedRepository] = useState<string | null>(null);
  const [stageIndex, setStageIndex] = useState(0);

  const chromeFor = (stepIndex: number): FirstRunChrome => ({
    displayName: firstName,
    email,
    onLogOut,
    stepIndex,
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
        githubConnected={githubConnected}
        chrome={chromeFor(1)}
        onConnectGitHub={() => {
          setGithubConnected(true);
          setScreen("choose");
        }}
        onContinue={() => setScreen("choose")}
      />
    );
  }

  if (screen === "choose") {
    return (
      <FirstRunChooseScreen
        repositories={FIRST_RUN_REPOSITORIES}
        selectedRepository={selectedRepository}
        chrome={chromeFor(2)}
        onSelectRepository={setSelectedRepository}
        onEditConnection={() => setScreen("connect")}
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
        chrome={chromeFor(3)}
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

  return board ?? <FirstRunBoardExit />;
}
