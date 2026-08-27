export type FirstRunScreenId = "welcome" | "connect" | "choose" | "tickets" | "analysis" | "board";

export type FirstRunChrome = {
  email?: string;
  displayName?: string;
  onLogOut?: () => void;
  stepIndex: number;
};

export type FirstRunTicketSource = "github-issues" | "jira" | "linear";

export type FirstRunAnalysisStatus = "running" | "overrun" | "failed";

export type FirstRunScoredTicket = {
  id: string;
  title: string;
  source: string;
  confidenceScore: number;
};
