/// <reference types="jest" />
import {
  buildAuthHeaders,
  resolveApiBaseUrl,
  launchDarklyConnection,
  type LaunchDarklyConnectionProps,
} from './auth';

describe('LaunchDarkly Auth Module', () => {
  describe('buildAuthHeaders', () => {
    it('should build auth headers with valid connection', () => {
      const connection: LaunchDarklyConnectionProps = {
        apiAccessToken: 'test-token-12345',
      };

      const headers = buildAuthHeaders(connection);

      expect(headers).toEqual({
        Authorization: 'test-token-12345',
        'Content-Type': 'application/json',
      });
    });

    it('should throw error when connection is undefined', () => {
      const connection = undefined as any;

      expect(() => buildAuthHeaders(connection)).toThrow(
        /Connection is required/i,
      );
    });

    it('should throw error when apiAccessToken is missing', () => {
      const connection: LaunchDarklyConnectionProps = {
        apiAccessToken: '',
      };

      expect(() => buildAuthHeaders(connection)).toThrow(
        /Missing or empty API access token/i,
      );
    });

    it('should throw error when apiAccessToken is whitespace only', () => {
      const connection: LaunchDarklyConnectionProps = {
        apiAccessToken: '   ',
      };

      expect(() => buildAuthHeaders(connection)).toThrow(
        /Missing or empty API access token/i,
      );
    });

    it('should handle tokens with special characters', () => {
      const connection: LaunchDarklyConnectionProps = {
        apiAccessToken: 'token_with-special.chars_123',
      };

      const headers = buildAuthHeaders(connection);

      expect(headers.Authorization).toBe('token_with-special.chars_123');
    });
  });

  describe('resolveApiBaseUrl', () => {
    it('should return default US API endpoint when not configured', () => {
      expect(resolveApiBaseUrl(undefined)).toBe(
        'https://app.launchdarkly.com/api/v2',
      );
      expect(resolveApiBaseUrl({ apiAccessToken: 'token' })).toBe(
        'https://app.launchdarkly.com/api/v2',
      );
    });

    it('should normalize EU host by appending /api/v2', () => {
      const connection: LaunchDarklyConnectionProps = {
        apiAccessToken: 'token',
        apiBaseUrl: 'https://app.eu.launchdarkly.com',
      };

      expect(resolveApiBaseUrl(connection)).toBe(
        'https://app.eu.launchdarkly.com/api/v2',
      );
    });

    it('should keep custom value when already ending with /api/v2', () => {
      const connection: LaunchDarklyConnectionProps = {
        apiAccessToken: 'token',
        apiBaseUrl: 'https://app.eu.launchdarkly.com/api/v2/',
      };

      expect(resolveApiBaseUrl(connection)).toBe(
        'https://app.eu.launchdarkly.com/api/v2',
      );
    });
  });

  describe('launchDarklyConnection', () => {
    it('should have correct configuration', () => {
      expect(launchDarklyConnection.name).toBe('LaunchDarkly');
      expect(launchDarklyConnection.description).toContain('REST API');
      expect(launchDarklyConnection.props).toBeDefined();
    });

    it('should mark apiAccessToken as required and sensitive', () => {
      const tokenProp = (launchDarklyConnection as any).props.apiAccessToken;

      expect(tokenProp.required).toBe(true);
      expect((tokenProp as any).sensitive).toBe(true);
    });

    it('should expose optional apiBaseUrl setting', () => {
      const baseUrlProp = (launchDarklyConnection as any).props.apiBaseUrl;

      expect(baseUrlProp).toBeDefined();
      expect(baseUrlProp.required).toBe(false);
    });
  });
});
