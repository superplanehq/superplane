import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import type { SplitRunPhaseStatus, SplitRunStreamLine } from "./splitRunMocks";

const ANALYSIS_WAIT_COMMAND = "Wait for the next message";

export function withCanceledAnalysisWait(
  note: SplitRunStreamLine,
  section: CommandSection,
  analysisStatus?: SplitRunPhaseStatus,
  nodeStatus?: SplitRunPhaseStatus,
): SplitRunStreamLine {
  if (!isCanceledAnalysisWait(section, analysisStatus, nodeStatus)) {
    return note;
  }
  return {
    ...note,
    status: "cancelled",
    promptStatus: note.promptStatus === "failed" ? "cancelled" : note.promptStatus,
  };
}

function isCanceledAnalysisWait(
  section: CommandSection,
  analysisStatus?: SplitRunPhaseStatus,
  nodeStatus?: SplitRunPhaseStatus,
): boolean {
  if (section.status !== "failed" || section.text.trim() !== ANALYSIS_WAIT_COMMAND) {
    return false;
  }
  if (analysisStatus === "cancelled") {
    return true;
  }
  return analysisStatus === "passed" && nodeStatus === "cancelled";
}
