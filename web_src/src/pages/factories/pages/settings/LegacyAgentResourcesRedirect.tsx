import { Navigate, useSearchParams } from "react-router";

export function LegacyAgentResourcesRedirect() {
  const [searchParams] = useSearchParams();
  const target = searchParams.get("tab") === "skills" ? "../workspace/skills" : "../workspace/mcp";
  return <Navigate to={target} replace />;
}
