import { Navigate, useSearchParams } from "react-router";
import { getWelcomeRedirectPath } from "./welcomeRedirectPath";

export default function WelcomeRedirect() {
  const [searchParams] = useSearchParams();
  return <Navigate to={getWelcomeRedirectPath(searchParams.get("redirect"))} replace />;
}
