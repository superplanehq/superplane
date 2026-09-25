export const DEPENDABOT_INTAKE_SETUP_COPY = {
  pageTitle: "Which Dependabot alerts should create tasks?",
  helper: (repository: string, skipInitialImport: boolean) =>
    skipInitialImport
      ? `SuperPlane listens for new Dependabot alerts from ${repository}.`
      : `SuperPlane imports open Dependabot alerts from ${repository} and listens for new alerts.`,
  repositorySection: "Repository",
  repositoryFallback: "Workspace backlog repository",
  create: "Create intake",
  creating: "Creating intake...",
  createError: "SuperPlane could not create the Dependabot intake.",
} as const;
