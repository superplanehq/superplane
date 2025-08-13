import { useParams } from 'react-router-dom';
import { Options } from '../api-client/sdk.gen';

// Hook to get current organization ID
export const useOrganizationId = (): string | null => {
  const { organizationId } = useParams<{ organizationId: string }>();
  return organizationId || null;
};

// Function to extract organization ID from current URL
const getOrganizationIdFromUrl = (): string | null => {
  const pathSegments = window.location.pathname.split('/');
  
  // Check if we're in the /app/:organizationId route pattern
  if (pathSegments[1] === 'app' && pathSegments[2]) {
    return pathSegments[2];
  }
  
  // Check if we're in the /:organizationId route pattern (for settings, canvas, etc.)
  if (pathSegments[1] && pathSegments[1] !== 'auth' && pathSegments[1] !== 'login' && pathSegments[1] !== 'register') {
    return pathSegments[1];
  }
  
  return null;
};

// Helper function to add organization header to API options
export const withOrganizationHeader = <T>(options: Options<T> = {} as Options<T>): Options<T> => {
  const organizationId = getOrganizationIdFromUrl();
  
  const headers: Record<string, string> = {
    ...options.headers,
  };
  
  if (organizationId) {
    headers['x-organization-id'] = organizationId;
  }
  
  return {
    ...options,
    headers,
  };
};

// Helper function for making direct API calls with organization context
export const apiCall = async (url: string, options: RequestInit = {}): Promise<Response> => {
  const organizationId = getOrganizationIdFromUrl();
  
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };
  
  if (organizationId) {
    headers['x-organization-id'] = organizationId;
  }
  
  return fetch(url, {
    ...options,
    credentials: 'include',
    headers,
  });
};