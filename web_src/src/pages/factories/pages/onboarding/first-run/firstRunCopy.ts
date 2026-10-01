import { GITHUB_INSTALL_REQUEST_NEXT, githubInstallRequestBody } from "@/lib/githubInstallRequestCopy";

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
    connectAction: "Connect GitHub",
    stepRepository: "Choose repository",
    connectError: "SuperPlane could not connect to GitHub. Check your access and try again.",
    openingGitHub: "Opening GitHub…",
    loadingAccounts: "Loading GitHub access…",
  },
  choose: {
    headline: "Choose your first repository",
    switchAccount: "Switch account",
    connectAnotherAccount: "Connect another GitHub account",
    repositoryLabel: "Repository",
    repositoryHelper: "SuperPlane scores tickets against this codebase and opens pull requests here for review.",
    searchPlaceholder: "Search repositories",
    missingRepository: "Do not see your repository?",
    grantAccess: "Grant access on GitHub",
    openingGitHub: "Opening GitHub…",
    synchronizing: "Repositories appear as GitHub verifies access…",
    installRequested: (organization: string) => `Waiting for approval for ${organization}.`,
    installRequestedBody: githubInstallRequestBody,
    installRequestedNext: GITHUB_INSTALL_REQUEST_NEXT,
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
    jiraHelper: "Connect Jira and choose a project.",
    jiraSoonHelper: "Jira intake is not available yet.",
    jiraProjectHeading: "Choose a Jira project",
    jiraLookupLoading: "Checking whether Jira is available…",
    jiraLookupFailed: "SuperPlane could not confirm that Jira is available. Choose GitHub Issues to continue.",
    linearHelper: "Scan your Linear backlog.",
    analyze: "Scan my backlog",
    continue: "Continue",
    saving: "Saving ticket source…",
  },
  agent: {
    headline: "Connect agent",
    modelSourceBody: "Choose the models that run your agents.",
    ownKeyBody: "Connect Anthropic, OpenAI, or OpenRouter. SuperPlane uses this key for agent runs.",
    modelSourceHeading: "Choose the model source",
    ownKey: "Your key",
    ownKeyHelper: "Connect Anthropic, OpenAI, or OpenRouter.",
    hostedModels: "SuperPlane-hosted models",
    hostedModelsHelper: "SuperPlane runs the agents. You do not connect a provider.",
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
    stageScoring: "Agent review started. Open the board to review and start work.",
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
