import { Navigate } from "react-router";

/** Legacy route: skills list lives on the Agent settings page. */
export function FactorySettingsSkillsPage() {
  return <Navigate to="../agent" replace />;
}
