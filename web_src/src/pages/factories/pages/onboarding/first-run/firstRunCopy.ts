/** Copy for the first-run flow. Source: docs/prd/onboarding-first-run.md */

export const FIRST_RUN_COPY = {
  chrome: {
    logOut: "Log out",
    loggedInAs: "Logged in as",
    stepLabel: (step: number, total: number) => `Step ${step} of ${total}`,
  },
  welcome: {
    greeting: (firstName: string) => `Hi ${firstName}.`,
    headline: "See what SuperPlane can ship from your backlog",
    intro: "Each ticket is scored by how confident SuperPlane is that an agent can complete it.",
    getStarted: "Get started",
  },
  connect: {
    headline: "Connect GitHub",
    body: "SuperPlane reads your repositories. It does not start work yet.",
    trust: "SuperPlane does not change code without your approval on a specific ticket.",
    connectGitHub: "Connect GitHub",
    connected: "Connected",
    continue: "Choose a repository",
    connectError: "SuperPlane could not connect to GitHub. Check your access and try again.",
  },
  choose: {
    headline: "Choose a repository",
    repositoryLabel: "Repository",
    repositoryHelper: "Select the repository you want SuperPlane to analyze.",
    searchPlaceholder: "Search repositories",
    missingRepository: "Do not see your repository?",
    editConnection: "Edit the GitHub connection.",
    continue: "Choose a repository to continue",
    moreLater: "You can add more repositories later.",
  },
  tickets: {
    headline: "Connect your ticket system",
    trust: "SuperPlane does not change any tickets.",
    scoreHint:
      "It only analyzes them and shows a confidence score for how likely SuperPlane is to address each ticket.",
    githubIssues: "GitHub Issues",
    jira: "Jira",
    linear: "Linear",
    githubIssuesHelper: "Uses GitHub Issues on this repository. No extra setup.",
    jiraHelper: "Find tickets in your Jira backlog.",
    linearHelper: "Find tickets in your Linear backlog.",
    analyze: "Analyze my tickets",
  },
  analysis: {
    headline: "Analyzing your backlog",
    body: "SuperPlane reads your code and your tickets. This takes a few minutes.",
    reassurance: "Nothing is changed. No work starts.",
    stage1: "Reading the repository structure",
    stage2: "Reading open tickets",
    stage3: "Scoring each ticket against the codebase",
    leaveHint: "You can leave this page. SuperPlane opens the board when the analysis finishes.",
    overrun: "The analysis needs more time than usual. It is still running.",
    failure: "The analysis did not finish. Try again.",
    retry: "Run analysis again",
  },
  results: {
    headline: "Tickets SuperPlane can implement",
    subhead: "Each score shows how confident SuperPlane is that an agent can complete the ticket correctly.",
    approve: "Approve",
    approved: "Approved",
    helper: "Work starts only on tickets you approve.",
    empty: "No tickets scored above 65%. Connect more of your backlog or create a work order yourself.",
    rescan: "Rescan backlog",
  },
  board: {
    backlogHintTitle: "Tickets land here first.",
    backlogHintBody: "Intake is scoring issues now. Review the ones SuperPlane can implement, then start the work.",
  },
} as const;

export const FIRST_RUN_STAGES = [
  FIRST_RUN_COPY.analysis.stage1,
  FIRST_RUN_COPY.analysis.stage2,
  FIRST_RUN_COPY.analysis.stage3,
] as const;
