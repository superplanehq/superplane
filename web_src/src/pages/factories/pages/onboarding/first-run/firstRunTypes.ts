export type FirstRunScreenId = "welcome" | "connect" | "choose" | "tickets" | "analysis" | "board";

export type FirstRunChrome = {
  email?: string;
  displayName?: string;
  avatarUrl?: string | null;
  /** Organization the workspace under setup belongs to, for the account menu. */
  organizationId: string;
  organizationName: string;
  /** Workspace key, so the account menu's settings links resolve under it. */
  factoryKey?: string;
  /**
   * Set when the user has somewhere to go: another workspace in this
   * organization, or another organization entirely. Shows "Quit onboarding"
   * in the account menu, which deletes the placeholder workspace and
   * navigates away instead of leaving the user stuck in setup. When unset,
   * the account menu still offers Sign out.
   */
  onQuitOnboarding?: () => void;
  stepIndex: number;
  /** Number of step dots. Set it when the flow skips a screen. */
  stepCount?: number;
  /** Storybook only: opens the account menu on mount, for design review. */
  menuDefaultOpen?: boolean;
};

export type FirstRunTicketSource = "github-issues" | "jira" | "linear";

export type FirstRunAnalysisStatus = "running" | "overrun" | "failed";

export type FirstRunScoredTicket = {
  id: string;
  title: string;
  source: string;
  confidenceScore: number;
};
