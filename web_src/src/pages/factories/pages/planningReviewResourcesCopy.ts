export const PLANNING_REVIEW_RESOURCES_COPY = {
  title: "Resources",
  kindMcp: "MCP server",
  kindSkill: "Skill",
  settingsLink: "workspace settings page",
  manage: "Manage",
  offForWorkspace: "Off for the workspace.",
  loading: "Loading resources…",
  loadError: "SuperPlane could not load resources. Try again.",
  enableLabel: (name: string) => `Enable ${name}`,
} as const;
