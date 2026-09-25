export const GITHUB_INTAKE_SETUP_COPY = {
  pageTitle: "How should GitHub issues become tasks?",
  headline: "Which repository should create tasks?",
  helper: "SuperPlane listens for new issues.",
  repositoryLabel: "Repository",
  importExistingHelper: "SuperPlane adds the newest open issues from this repository.",
  importExistingHelperOff: "SuperPlane does not import existing issues.",
  wizardFinish: "Finish",
  wizardFinishing: "Finishing...",
  wizardCreateError: "SuperPlane could not create the GitHub intake.",
  repositoryMissing:
    "No backlog repository is set in this workspace. SuperPlane still creates the intake. Connect a repository on the intake canvas before it can receive issues.",
} as const;
