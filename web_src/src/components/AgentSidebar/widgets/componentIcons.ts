/**
 * Component name → fallback icon slug, mirroring what each canvas mapper resolves to when the
 * server-supplied `componentDefinition.icon` is missing. Keeping this map in sync with the
 * mappers ensures NodeChip pills render the same icon the user sees on the canvas card itself.
 *
 * Search "iconSlug:" under web_src/src/pages/app/mappers when adding new built-in
 * components here.
 */
export const BUILTIN_COMPONENT_ICON_SLUGS: Record<string, string> = {
  noop: "circle-off",
  display: "monitor",
  addMemory: "database",
  deleteMemory: "database",
  readMemory: "database",
  updateMemory: "database",
  upsertMemory: "database",
  if: "split",
  http: "globe",
  graphql: "network",
  ssh: "terminal",
  runner: "terminal",
  runnerJS: "code",
  runnerPython: "code",
  runnerBash: "code",
  runnerClaudeCode: "code",
  timeGate: "clock",
  filter: "filter",
  wait: "clock",
  approval: "hand",
  merge: "git-merge",
  schedule: "calendar-clock",
  webhook: "webhook",
  start: "play",
};
