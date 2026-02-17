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

  /**
   * Optional custom LaunchDarkly API host (for example EU region).
   *
   * @example "https://app.eu.launchdarkly.com"
   * @defaultValue "https://app.launchdarkly.com/api/v2"
   */
  apiBaseUrl?: string;
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

/**
 * Resolve the LaunchDarkly API base URL for the active connection.
 *
 * If a custom host is provided, this helper ensures the final value ends with
 * `/api/v2`. Otherwise, the default US endpoint is used.
 */
export function resolveApiBaseUrl(
  connection?: LaunchDarklyConnectionProps,
): string {
  const configured = connection?.apiBaseUrl?.trim();
  if (!configured) {
    return LD_API_BASE_URL;
  }

  const normalized = configured.replace(/\/+$/, '');
  if (normalized.endsWith('/api/v2')) {
    return normalized;
  }

  return `${normalized}/api/v2`;
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
    apiBaseUrl: {
      type: 'string',
      label: 'API Base URL (Optional)',
      description:
        'Custom LaunchDarkly API host for regional tenants, e.g. https://app.eu.launchdarkly.com. If omitted, uses https://app.launchdarkly.com/api/v2.',
      required: false,
    },
  },
});
