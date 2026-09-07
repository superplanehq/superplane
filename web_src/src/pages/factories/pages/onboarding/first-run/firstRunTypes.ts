export type FirstRunScreenId = "welcome" | "connect" | "choose" | "tickets" | "analysis" | "board";

export type FirstRunChrome = {
  email?: string;
  displayName?: string;
  onLogOut?: () => void;
  /**
   * Set when the user has another workspace or another organization to
   * open. The shell shows the organization switch next to Log out.
   */
  organizationSwitch?: { currentOrganizationRouteId: string };
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
