export const DEPENDABOT_INTAKE_SETUP_COPY = {
  pageTitle: "Which Dependabot alerts should create tasks?",
  helper: (repository: string) => `SuperPlane listens for new Dependabot alerts from ${repository}.`,
  importExistingHelper: "SuperPlane imports open Dependabot alerts from this repository.",
  importExistingHelperOff: "SuperPlane does not import existing alerts.",
  repositorySection: "Repository",
  repositoryFallback: "Workspace backlog repository",
  setupRequired: "Connect GitHub and select a backlog repository in workspace setup before you create this intake.",
  create: "Create intake",
  creating: "Creating intake...",
  createError: "SuperPlane could not create the Dependabot intake.",
} as const;
