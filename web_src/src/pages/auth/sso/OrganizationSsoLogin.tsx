import { useEffect } from "react";
import { useParams, useSearchParams } from "react-router";

export function OrganizationSsoLogin() {
  const { slug } = useParams<{ slug: string }>();
  const [params] = useSearchParams();

  useEffect(() => {
    if (!slug) {
      return;
    }
    const redirect = params.get("redirect");
    const query = redirect ? `?redirect=${encodeURIComponent(redirect)}` : "";
    window.location.replace(`/auth/sso/organizations/${encodeURIComponent(slug)}/start${query}`);
  }, [params, slug]);

  return <p className="p-6 text-sm">Continue to your identity provider.</p>;
}
