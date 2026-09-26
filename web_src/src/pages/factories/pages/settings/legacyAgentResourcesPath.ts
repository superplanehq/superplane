export function legacyAgentResourcesPath(searchParams: URLSearchParams): string {
  const skills = searchParams.get("tab") === "skills";
  const add = searchParams.get("dialog") === "add";
  if (skills) {
    return add ? "../workspace/skills/new" : "../workspace/skills";
  }
  return add ? "../workspace/mcp?dialog=add" : "../workspace/mcp";
}
