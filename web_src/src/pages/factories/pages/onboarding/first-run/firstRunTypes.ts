export type FirstRunScreenId = "welcome" | "connect" | "choose" | "tickets" | "analysis" | "board";

export type FirstRunWorkspaceOption = {
  id?: string;
  key?: string;
  name?: string;
  lines?: Array<{ id?: string }> | null;
};

export type FirstRunChrome = {
  busy?: boolean;
  email?: string;
  displayName?: string;
  onLogOut?: () => void;
  /**
   * Set when the user belongs to another organization. The shell shows
   * the organization switch next to Log out.
   */
  organizationSwitch?: { currentOrganizationRouteId: string };
  /**
   * Set when another workspace exists in this organization. The shell
   * shows the workspace initials in the bottom-left corner.
   */
  workspaceSwitch?: {
    organizationId: string;
    currentFactoryId: string;
    factories: FirstRunWorkspaceOption[];
  };
  stepIndex: number;
  /** Number of step dots. Set it when the flow skips a screen. */
  stepCount?: number;
  onBack?: () => void;
};

export type FirstRunTicketSource = "github-issues" | "jira" | "linear";

export type FirstRunAnalysisStatus = "running" | "overrun" | "failed";

export type FirstRunScoredTicket = {
  id: string;
  title: string;
  source: string;
  confidenceScore: number;
};
