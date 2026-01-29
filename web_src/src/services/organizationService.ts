/**
 * Organization service for non-API organization endpoints
 * Handles listing and creating organizations
 *
 * Note: For /api/v1/organizations/* endpoints, use the generated SDK in @/api-client
 */

export interface Organization {
  id: string;
  name: string;
  canvasCount?: number;
  memberCount?: number;
}

export interface CreateOrganizationParams {
  name: string;
}

/**
 * Fetch user's organizations
 */
export async function fetchOrganizations(): Promise<Organization[]> {
  const response = await fetch("/organizations", {
    method: "GET",
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to load organizations");
  }

  return response.json();
}

/**
 * Create a new organization
 */
export async function createOrganization(
  params: CreateOrganizationParams
): Promise<Organization> {
  const response = await fetch("/organizations", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify({
      name: params.name.trim(),
    }),
  });

  if (!response.ok) {
    let errorMessage = "Failed to create organization";

    try {
      const errorData = await response.json();
      errorMessage = errorData.message || errorMessage;
    } catch {
      if (response.status === 409) {
        errorMessage = "An organization with this name already exists";
      } else {
        errorMessage = `Failed to create organization (${response.status})`;
      }
    }

    throw new Error(errorMessage);
  }

  return response.json();
}
