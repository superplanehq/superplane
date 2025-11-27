// Setup function to add 401 redirect interceptor to the global API client
let interceptorSetup = false;

export const setupApiInterceptor = (): void => {
  if (interceptorSetup) return;

  // Skip setup in test environments to avoid conflicts
  if (typeof window !== 'undefined' && window.location.hostname === '127.0.0.1') {
    interceptorSetup = true;
    return;
  }

  const originalFetch = globalThis.fetch;

  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const response = await originalFetch(input, init);

    if (response.status === 401 && isApiRequest(input)) {
      const currentPath = window.location.pathname + window.location.search;
      const redirectUrl = encodeURIComponent(currentPath);

      window.location.href = `/login?redirect=${redirectUrl}`;

      throw new Error('Unauthorized - redirecting to login');
    }

    return response;
  };

  interceptorSetup = true;
};

function isApiRequest(input: RequestInfo | URL): boolean {
  const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;

  return url.includes('/api/');
}