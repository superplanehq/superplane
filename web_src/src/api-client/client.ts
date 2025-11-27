// Setup function to add 401 redirect interceptor to the global API client
let interceptorSetup = false;

export const setupApiInterceptor = (): void => {
  if (interceptorSetup) return;

  const originalFetch = globalThis.fetch;

  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const response = await originalFetch(input, init);

    if (response.status === 401 && isApiRequest(input)) {
      window.location.href = `/`;

      throw new Error("Unauthorized");
    }

    return response;
  };

  interceptorSetup = true;
};

function isApiRequest(input: RequestInfo | URL): boolean {
  const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;

  return url.includes("/api/");
}
