import type { ReactNode } from "react";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function RequireBYOKAccess({ children }: { children: ReactNode }) {
  const { organizationId } = useFactorySettingsLayout();
  const access = useBYOKAccess(organizationId);
  if (access.isLoading) return <p role="status">Loading model access…</p>;
  if (!access.ready)
    return <p role="alert">SuperPlane could not verify model access. Reload this page to try again.</p>;
  if (!access.allowed)
    return (
      <p role="alert">Your organization does not have access to provider credentials. Contact your administrator.</p>
    );
  return children;
}
