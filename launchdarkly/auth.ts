import { createConnection } from '@superplane/framework';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/** LaunchDarkly REST API v2 base URL. */
export const LD_API_BASE_URL = 'https://app.launchdarkly.com/api/v2';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Connection properties required to authenticate with the LaunchDarkly API.
 *
 * @remarks
 * Uses a personal or service **API access token** issued from the
 * LaunchDarkly Authorization settings page.
 *
 * @see {@link https://launchdarkly.com/docs/home/account/api | LaunchDarkly – API Access Tokens}
 */
export interface LaunchDarklyConnectionProps {
  /**
   * A LaunchDarkly API access token (personal or service token).
   *
   * The token must have sufficient permissions for the operations the
   * integration will perform (e.g. `reader` for GET, `writer` for DELETE).
   */
  apiAccessToken: string;
}

/**
 * Standard headers sent with every authenticated request to the
 * LaunchDarkly REST API.
 */
export interface LaunchDarklyAuthHeaders {
  Authorization: string;
  'Content-Type': string;
  [key: string]: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Build the HTTP headers required for an authenticated LaunchDarkly API call.
 *
 * @param connection - The active connection containing the API access token.
 * @returns A headers object suitable for use with `fetch` / `axios` / etc.
 *
 * @example
 * ```ts
 * const headers = buildAuthHeaders(connection);
 * const res = await fetch(`${LD_API_BASE_URL}/flags/my-project`, { headers });
 * ```
 */
/**
 * Validates a LaunchDarkly connection object.
 *
 * @param connection - The connection to validate.
 * @throws If the connection is missing required properties.
 */
function validateConnection(
  connection: LaunchDarklyConnectionProps | undefined,
): asserts connection {
  if (!connection) {
    throw new Error(
      '[LaunchDarkly] Connection is required but was not provided. ' +
        'Ensure the connection is properly resolved by the runtime.',
    );
  }

  if (!connection.apiAccessToken?.trim()) {
    throw new Error(
      '[LaunchDarkly] Missing or empty API access token. ' +
        'Please provide a valid token in the connection settings.',
    );
  }
}

export function buildAuthHeaders(
  connection: LaunchDarklyConnectionProps,
): LaunchDarklyAuthHeaders {
  validateConnection(connection);

  return {
    Authorization: connection.apiAccessToken,
    'Content-Type': 'application/json',
  };
}

// ---------------------------------------------------------------------------
// Connection definition
// ---------------------------------------------------------------------------

/**
 * SuperPlane connection definition for LaunchDarkly.
 *
 * Provides the authentication configuration that all components in this
 * integration share. End-users supply their LaunchDarkly API access token
 * when setting up the connection in the SuperPlane UI.
 *
 * @see {@link https://launchdarkly.com/docs/home/account/api | LaunchDarkly – API Access Tokens}
 */
export const launchDarklyConnection = createConnection<LaunchDarklyConnectionProps>({
  name: 'LaunchDarkly',
  description:
    'Connects to the LaunchDarkly REST API using an API access token.',
  props: {
    apiAccessToken: {
      type: 'string',
      label: 'API Access Token',
      description:
        'A personal or service API access token generated from the LaunchDarkly Authorization page.',
      required: true,
      sensitive: true, // masks the value in the UI
    },
  },
});
