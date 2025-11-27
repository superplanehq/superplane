// Setup function to add 401 redirect interceptor to the global API client
let interceptorSetup = false;

export const setupApiInterceptor = (): void => {
  if (interceptorSetup) return;

  const originalFetch = globalThis.fetch;

  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const response = await originalFetch(input, init);

    if (response.status === 401) {
      const currentPath = window.location.pathname + window.location.search;
      const redirectUrl = encodeURIComponent(currentPath);

      window.location.href = `/login?redirect=${redirectUrl}`;

      throw new Error('Unauthorized - redirecting to login');
    }

    return response;
  };

  interceptorSetup = true;
};