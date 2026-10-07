import { Navigate } from "react-router";

/** Legacy route: MCP list and inbound clients moved to Agent and Connect. */
export function FactorySettingsMCPPage() {
  return <Navigate to="../agent" replace />;
}
