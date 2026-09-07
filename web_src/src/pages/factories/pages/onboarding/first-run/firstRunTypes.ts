export type FirstRunScreenId = "welcome" | "connect" | "choose" | "tickets" | "analysis" | "board";

export type FirstRunChrome = {
  email?: string;
  displayName?: string;
  onLogOut?: () => void;
  /**
   * Set when the user has somewhere to go on cancel: another workspace in
   * this organization, or another organization entirely. The shell shows a
   * close (X) control instead of "Log out", and it cancels setup rather than
   * signing the user out. Mutually exclusive with `onLogOut`.
   */
  onCancel?: () => void;
  stepIndex: number;
  /** Number of step dots. Set it when the flow skips a screen. */
  stepCount?: number;
};

export type FirstRunTicketSource = "github-issues" | "jira" | "linear";

export type FirstRunAnalysisStatus = "running" | "overrun" | "failed";

export type FirstRunScoredTicket = {
  id: string;
  title: string;
  source: string;
  confidenceScore: number;
};
