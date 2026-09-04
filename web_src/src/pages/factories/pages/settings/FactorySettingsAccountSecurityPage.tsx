import { Navigate, useLocation } from "react-router";

export function FactorySettingsAccountSecurityPage() {
  const { pathname, search } = useLocation();
  return <Navigate to={`${pathname.replace(/\/security\/?$/, "/profile")}${search}`} replace />;
}
