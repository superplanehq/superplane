import { isUUID } from "@/lib/utils";
import { useParams } from "react-router-dom";

export const useOrganizationId = (): string | null => {
  const { organizationId } = useParams<{ organizationId: string }>();
  if (organizationId && !isUUID(organizationId)) {
    return null;
  }
  return organizationId || null;
};

const getOrganizationIdFromUrl = (): string | null => {
  const pathSegments = window.location.pathname.split("/");

  // Check if we're in the /:organizationId route pattern
  if (pathSegments[1] === "" && pathSegments[2]) {
    return pathSegments[2];
  }

  // Check if we're in the /:organizationId route pattern (for settings, canvas, etc.)
  if (pathSegments[1] && pathSegments[1] !== "auth" && pathSegments[1] !== "login" && pathSegments[1] !== "register") {
    return pathSegments[1];
  }

  return null;
};

export function withOrganizationHeader(options: any = {}): any {
  const organizationId = getOrganizationIdFromUrl();

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

  return {
    ...options,
    headers,
  };
}
