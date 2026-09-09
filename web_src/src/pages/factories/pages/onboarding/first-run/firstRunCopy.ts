import {
  GITHUB_INSTALL_REQUEST_NEXT,
  GITHUB_INSTALL_REQUEST_TITLE,
  githubInstallRequestBody,
} from "@/lib/githubInstallRequestCopy";

function ticketCount(count: number) {
  return count === 1 ? "1 ticket" : `${count} tickets`;
}

export const FIRST_RUN_COPY = {
  chrome: {
    logOut: "Log out",
    switchOrganization: "Switch organization",
    switchWorkspace: "Switch workspace",
    loggedInAs: "Logged in as",
    stepLabel: (step: number, total: number) => `Step ${step} of ${total}`,
    back: "Back",
  },
  welcome: {
    greeting: (firstName: string) => `Hi ${firstName}.`,
    headline: "One-shot the routine work in your backlog",
    intro:
      "SuperPlane finds the tickets agents can finish with high confidence and turns them into review-ready pull requests.",
    getStarted: "Get started",
  },
  connect: {
    headline: "Connect SuperPlane to GitHub",
    body: "SuperPlane reads your code and tickets, and returns finished work as pull requests.",
    signedInAs: (login: string) => `You are signed in to GitHub as ${login}.`,
    connectGitHub: "Connect GitHub",
    connectAction: "Connect",
    stepConnected: "Connected to GitHub",
    stepOrganization: "Choose organization",
    stepOrganizationDone: (organization: string) => `Organization: ${organization}`,
    stepRepository: "Choose repository",
    selectAccount: "Choose a GitHub organization",
    selectAccountBody: "SuperPlane sees only the repositories and tickets in the organization you choose.",
    useAccount: (account: string) => `Use ${account}`,
    missingAccount: "Do not see your GitHub account or organization?",
    installThere: "Install SuperPlane on more GitHub organizations.",
    connectError: "SuperPlane could not connect to GitHub. Check your access and try again.",
    installRequested: GITHUB_INSTALL_REQUEST_TITLE,
    installRequestedBody: githubInstallRequestBody,
    installRequestedNext: GITHUB_INSTALL_REQUEST_NEXT,
    openingGitHub: "Opening GitHub…",
    loadingAccounts: "Loading GitHub accounts…",
    connectingAccount: (account: string) => `Connecting ${account}…`,
  },
  choose: {
    headline: "Choose your first repository",
    repositoryLabel: "Repository",
    repositoryHelper: "SuperPlane scores tickets against this codebase and opens pull requests here for review.",
    searchPlaceholder: "Search repositories",
    missingRepository: "Do not see your repository?",
    editConnection: "Grant access on GitHub.",
    continue: "Choose a repository to continue",
    continueReady: "Continue",
    saving: "Saving repository…",
    loading: "Loading repositories…",
    moreLater: "You can add more repositories later.",
  },
  tickets: {
    headline: "Choose the backlog to scan",
    intro: "SuperPlane reads these tickets and scores each one by how likely an agent can complete it.",
    githubIssues: "GitHub Issues",
    jira: "Jira",
    linear: "Linear",
    githubIssuesHelper: "Issues on this repository. No extra setup.",
    jiraHelper: "Scan your Jira backlog.",
    linearHelper: "Scan your Linear backlog.",
    analyze: "Scan my backlog",
    continue: "Connect agent",
    saving: "Saving ticket source…",
  },
  agent: {
    headline: "Connect agent",
    loading: "Checking agent availability…",
  },
  // Shared by every screen that provisions the workspace. Hosted credentials
  // move that action from the agent screen to the ticket screen.
  finish: {
    action: "Finish setup",
    saving: "Finishing setup…",
  },
  analysis: {
    headline: "Your workspace is running",
    body: "SuperPlane scores each ticket for how likely an agent can one-shot it. High-confidence work is ready to run. Ambiguous work stays with your team.",
    stageImporting: "Importing your newest open tickets…",
    stageImported: (count: number, source = "your backlog") =>
      count === 1
        ? `Imported the newest open ticket from ${source}`
        : `Imported the ${count} newest open tickets from ${source}`,
    stageScoringPending: "Scoring tickets",
    stageScoring: "Started scoring. Open your board and run tickets as scores appear.",
    emptyImport: (source = "your backlog") => `We did not find any issues to import from ${source}`,
    emptyNext: "You can create a new task on your board.",
    stageScored: (total: number) => `${ticketCount(total)} scored`,
    readyCount: (count: number) => (count === 1 ? "1 ready to run" : `${count} ready to run`),
    goToBoard: "Go to your board",
    failure: "SuperPlane could not read the scoring progress. Your board still works.",
  },
  sphere: {
    phases: ["01 Plan", "02 Build", "03 Check", "04 Review"],
    discover: "Discover",
    verify: "Verify",
    awaitingCode: "Awaiting code",
    reviewReadyPr: "Review-ready PR",
    ticketsFound: (count: number) => `${count} tickets found`,
    captionSetup: "Awaiting setup",
    captionConnect: "Awaiting GitHub connection",
    captionOrganization: "Awaiting organization",
    captionRepository: (repository: string) => `Repository: ${repository}`,
    captionAwaitingRepository: "Awaiting repository",
    captionTickets: "Awaiting ticket source",
    captionScoring: (repository: string) => `Scoring: ${repository}`,
  },
  results: {
    headline: "Tickets SuperPlane can implement",
    subhead: "Each score shows how confident SuperPlane is that an agent can complete the ticket correctly.",
    approve: "Approve",
    approved: "Approved",
    helper: "Work starts only on tickets you approve.",
    empty: "No tickets scored above 65%. Connect more of your backlog or create a task yourself.",
    rescan: "Rescan backlog",
  },
  board: {
    backlogHintTitle: "Tickets land here first.",
    backlogHintBody:
      "New issues become tasks here. SuperPlane scores them for how well an agent can complete the work.",
  },
} as const;
