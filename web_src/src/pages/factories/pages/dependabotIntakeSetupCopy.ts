export const DEPENDABOT_INTAKE_SETUP_COPY = {
  pageTitle: "Which Dependabot alerts should create tasks?",
  helper: (repository: string) =>
    `SuperPlane listens for new Dependabot alerts from ${repository}. In the next step you choose which open alerts to import.`,
  repositorySection: "Repository",
  repositoryFallback: "Workspace backlog repository",
  setupRequired: "Connect GitHub and select a backlog repository in workspace setup before you create this intake.",
  create: "Create intake",
  creating: "Creating intake...",
  createError: "SuperPlane could not create the Dependabot intake.",
  import: {
    pageTitle: "Which open alerts should become tasks?",
    helper: "SuperPlane creates one task per package. New alerts for a package join its open task.",
    loading: "Loading open alerts...",
    loadError: "SuperPlane could not load the open alerts.",
    retry: "Try again",
    matchCount: (count: number) =>
      count === 1
        ? "1 package has open alerts that match your filters."
        : `${count} packages have open alerts that match your filters.`,
    empty: "No open alerts match your filters.",
    selectAll: "Select all",
    clearSelection: "Clear selection",
    importSelected: (count: number) => `Import selected (${count})`,
    importing: (done: number, total: number) => `Importing ${done} of ${total}...`,
    importError: "SuperPlane could not import a package. The imported tasks are in the Backlog.",
    skip: "Skip for now",
    caption: "Choosing alerts",
  },
} as const;
