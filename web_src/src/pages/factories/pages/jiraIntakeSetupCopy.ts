export const JIRA_INTAKE_SETUP_COPY = {
  pageTitle: "How should Jira issues become tasks?",
  wizardStepConnect: "Connect Jira",
  wizardStepConnectHelper:
    "Connect Jira so SuperPlane can read issues and receive new issue events. If SuperPlane already has this Jira site, open the integration and authorize it again.",
  wizardStepConnectExisting: "Choose the Jira site that SuperPlane will monitor.",
  wizardStepProject: "Which Jira project should create tasks?",
  wizardStepProjectHelper:
    "SuperPlane adds the 10 newest unresolved issues of this project. SuperPlane also listens for new issues.",
  wizardStepProjectHelperSkip: "SuperPlane listens for new issues. SuperPlane does not import existing issues.",
  wizardConnect: "Connect Jira",
  wizardConnecting: "Connecting...",
  wizardConnectError: "SuperPlane could not open Atlassian authorization.",
  wizardContinue: "Continue",
  wizardFinish: "Finish",
  wizardFinishing: "Finishing...",
  wizardProjectsLoading: "Loading Jira projects...",
  wizardProjectsError: "SuperPlane could not load Jira projects.",
  wizardProjectsEmpty: "This connection has no available projects.",
  wizardRetry: "Try again",
  wizardCreateError: "SuperPlane could not create the Jira intake.",
  wizardConnectionsLoading: "Loading Jira connections...",
} as const;
