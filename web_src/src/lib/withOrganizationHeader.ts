import { isReservedAppPathSegment } from "./reservedAppPaths";

const NON_ORGANIZATION_PATH_SEGMENTS = new Set(["auth", "register"]);

const getOrganizationIdFromUrl = (): string | null => {
  const pathSegments = window.location.pathname.split("/");
  const segment = pathSegments[1];

  // Check if we're in the /:organizationId route pattern (for settings, canvas, etc.)
  if (segment && !NON_ORGANIZATION_PATH_SEGMENTS.has(segment) && !isReservedAppPathSegment(segment)) {
    return segment;
  }

  return null;
};

type OrganizationHeaderInput = {
  organizationId?: string | null;
  headers?: HeadersInit;
};

export function withOrganizationHeader<TIn, TOut = TIn & { headers: Record<string, string> }>(
  options?: TIn & { organizationId?: string | null },
): TOut {
  const resolved = (options ?? {}) as TIn & OrganizationHeaderInput;
  // Prefer an explicit organizationId (e.g. from route params) over window.location
  // because window.location can be stale during router transitions.
  const organizationId = resolved.organizationId ?? getOrganizationIdFromUrl();

  const headers: Record<string, string> = {};

  if (resolved.headers) {
    if (resolved.headers instanceof Headers) {
      resolved.headers.forEach((value: string, key: string) => {
        headers[key] = value;
      });
    } else if (typeof resolved.headers === "object" && !Array.isArray(resolved.headers)) {
      Object.assign(headers, resolved.headers);
    }
  }

  if (organizationId) {
    headers["x-organization-id"] = organizationId;
  }

  // Avoid leaking our internal option into fetch/init objects.
  // Codegen clients ignore unknown top-level fields, but callers may also pass this to native fetch.
  const { organizationId: _ignored, headers: _ignoredHeaders, ...rest } = resolved;

  return {
    ...rest,
    headers,
  } as TOut;
}
