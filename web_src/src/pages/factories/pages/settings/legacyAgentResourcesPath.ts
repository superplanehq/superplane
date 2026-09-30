export function legacyAgentResourcesPath(searchParams: URLSearchParams) {
  const tab = searchParams.get("tab");
  const add = searchParams.get("dialog") === "add";
  if (tab === "skills") {
    return add ? "../workspace/skills/new" : "../workspace/agent";
  }
  return add ? "../workspace/agent?dialog=add" : "../workspace/agent";
}
