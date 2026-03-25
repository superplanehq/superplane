import { useParams } from "react-router-dom";

export const useOrganizationId = (): string | null => {
  const { organizationId } = useParams<{ organizationId: string }>();
  return organizationId || null;
};

const getOrganizationIdFromUrl = (): string | null => {
  const pathSegments = window.location.pathname.split("/");

  // Check if we're in the /:organizationId route pattern (for settings, canvas, etc.)
  if (pathSegments[1] && pathSegments[1] !== "auth" && pathSegments[1] !== "login" && pathSegments[1] !== "register") {
    return pathSegments[1];
  }

  return null;
};

export function withOrganizationHeader(options: any = {}): any {
  // Prefer an explicit organizationId (e.g. from route params) over window.location
  // because window.location can be stale during router transitions.
  const organizationId = options?.organizationId ?? getOrganizationIdFromUrl();

  const headers: Record<string, string> = {};

  if (options.headers) {
    if (options.headers instanceof Headers) {
      options.headers.forEach((value: string, key: string) => {
        headers[key] = value;
      });
    } else if (typeof options.headers === "object" && !Array.isArray(options.headers)) {
      Object.assign(headers, options.headers);
    }
  }

  if (organizationId) {
    headers["x-organization-id"] = organizationId;
  }

  // Avoid leaking our internal option into fetch/init objects.
  // Codegen clients ignore unknown top-level fields, but callers may also pass this to native fetch.
  const { organizationId: _ignored, ...rest } = options ?? {};

  return {
    ...rest,
    headers,
  };
}
