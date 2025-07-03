/// <reference lib="dom" />

declare module 'fetch-retry' {
  const _fetch: typeof fetch;

  type RequestDelayFunction = ((
    attempt: number,
    error: Error | null,
    response: Response | null
  ) => number);

  type RequestRetryOnFunction = ((
    attempt: number,
    error: Error | null,
    response: Response | null
  ) => boolean | Promise<boolean>);

  export interface RequestInitRetryParams {
    retries?: number;
    retryDelay?: number | RequestDelayFunction;
    retryOn?: number[] | RequestRetryOnFunction;
  }

  export type RequestInitWithRetry = RequestInit & RequestInitRetryParams;

  function fetchBuilder(fetch: typeof _fetch, defaults?: RequestInitRetryParams): ((input: Parameters<typeof _fetch>[0], init?: RequestInitWithRetry) => Promise<Response>);
  export default fetchBuilder;
}
