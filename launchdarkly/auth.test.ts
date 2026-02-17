/// <reference types="jest" />
import {
  buildAuthHeaders,
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
  });
});
