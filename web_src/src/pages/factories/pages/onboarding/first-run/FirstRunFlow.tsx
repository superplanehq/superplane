import { useEffect, useState, type ReactNode } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";
import { FirstRunBoardExit } from "./FirstRunBoardExit";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./FirstRunConnectScreen";
import { FIRST_RUN_REPOSITORIES, FIRST_RUN_STORY_EMAIL } from "./firstRunMocks";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";
import type { FirstRunAnalysisProgress } from "./firstRunAnalysisProgress";
import type { FirstRunChrome, FirstRunScreenId, FirstRunTicketSource } from "./firstRunTypes";
import { FirstRunWelcomeScreen } from "./FirstRunWelcomeScreen";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";

const STAGE_MS = 900;

/**
 * Clickable Storybook journey for the first-run PRD. Local state only.
 * Production wiring stays out of this file.
 */
export function FirstRunFlow({
  firstName,
  email = FIRST_RUN_STORY_EMAIL,
  initialScreen = "welcome",
  board,
  onLogOut,
}: {
  firstName?: string;
  email?: string;
  initialScreen?: FirstRunScreenId;
  board?: ReactNode;
  onLogOut?: () => void;
}) {
  const [screen, setScreen] = useState<FirstRunScreenId>(initialScreen);
  const [ticketSource, setTicketSource] = useState<FirstRunTicketSource | null>(null);
  const [selectedRepository, setSelectedRepository] = useState<string | null>(null);
  const [progress, setProgress] = useState<FirstRunAnalysisProgress>({ total: 12, scored: 0, ready: 0, stageIndex: 1 });

  const chromeFor = (stepIndex: number, onBack?: () => void): FirstRunChrome => ({
    displayName: firstName,
    email,
    onLogOut,
    stepIndex,
    onBack,
  });

  useEffect(() => {
    if (screen !== "analysis") return;
    const stageTimer = window.setInterval(() => {
      setProgress((current) => {
        const scored = Math.min(current.scored + 1, current.total);
        // Roughly two of three demo tickets score above the threshold.
        const ready = Math.ceil((scored * 2) / 3);
        return { total: current.total, scored, ready, stageIndex: scored === current.total ? 2 : 1 };
      });
    }, STAGE_MS);
    return () => window.clearInterval(stageTimer);
  }, [screen]);

  const analysisSphere: FirstRunSphereProps = {
    level: 1,
    animate: true,
    phasesLit: true,
    caption: selectedRepository ?? "",
    captionHighlight: "Scoring:",
    leftChip: {
      label: FIRST_RUN_COPY.sphere.discover,
      value: FIRST_RUN_COPY.sphere.ticketsFound(progress.total),
      tone: "amber",
    },
    rightChip: { label: FIRST_RUN_COPY.sphere.verify, value: FIRST_RUN_COPY.sphere.reviewReadyPr, tone: "amber" },
  };

  if (screen === "welcome") {
    return (
      <FirstRunWelcomeScreen firstName={firstName} chrome={chromeFor(0)} onGetStarted={() => setScreen("connect")} />
    );
  }

  if (screen === "connect") {
    return <FirstRunConnectScreen chrome={chromeFor(1)} onConnectGitHub={() => setScreen("choose")} />;
  }

  if (screen === "choose") {
    return (
      <FirstRunChooseScreen
        repositories={FIRST_RUN_REPOSITORIES}
        selectedRepository={selectedRepository}
        organizationName="acme"
        chrome={chromeFor(2, () => setScreen("connect"))}
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
        chrome={chromeFor(3, () => setScreen("choose"))}
        onSelectTicketSource={setTicketSource}
        onAnalyzeTickets={() => {
          if (!ticketSource) return;
          setProgress({ total: 12, scored: 0, ready: 0, stageIndex: 1 });
          setScreen("analysis");
        }}
      />
    );
  }

  if (screen === "analysis") {
    return (
      <FirstRunAnalysisScreen
        progress={progress}
        chrome={chromeFor(4)}
        sphere={analysisSphere}
        onGoToBoard={() => setScreen("board")}
      />
    );
  }

  return board ?? <FirstRunBoardExit />;
}
