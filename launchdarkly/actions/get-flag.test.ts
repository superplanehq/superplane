/// <reference types="jest" />
/// <reference types="node" />
import { getFlag } from '../actions/get-flag';
import type { LaunchDarklyConnectionProps } from '../auth';

describe('Get Feature Flag Action', () => {
  const mockConnection: LaunchDarklyConnectionProps = {
    apiAccessToken: 'test-token',
  };

  beforeEach(() => {
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('action definition', () => {
    it('should have correct metadata', () => {
      expect(getFlag.name).toBe('Get Feature Flag');
      expect(getFlag.description).toContain('Retrieve');
      expect(getFlag.connection).toBeDefined();
    });

    it('should have correct prop definitions', () => {
      expect((getFlag as any).props.projectKey).toBeDefined();
      expect((getFlag as any).props.projectKey.required).toBe(true);
      expect((getFlag as any).props.flagKey).toBeDefined();
      expect((getFlag as any).props.flagKey.required).toBe(true);
    });
  });

  describe('run method', () => {
    it('should throw error when projectKey is empty', async () => {
      const ctx = {
        props: { projectKey: '', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await expect(getFlag.run(ctx)).rejects.toThrow(
        'projectKey" is required and cannot be empty',
      );
    });

    it('should throw error when flagKey is empty', async () => {
      const ctx = {
        props: { projectKey: 'test-project', flagKey: '' },
        connection: mockConnection,
      } as any;

      await expect(getFlag.run(ctx)).rejects.toThrow(
        'flagKey" is required and cannot be empty',
      );
    });

    it('should fetch flag successfully', async () => {
      const mockFlag = {
        key: 'test-flag',
        name: 'Test Flag',
        kind: 'boolean' as const,
        _version: 1,
        creationDate: 1234567890,
        temporary: false,
        tags: ['test'],
        variations: [{ value: true }, { value: false }],
        archived: false,
        _links: {},
      };

      (global.fetch as any).mockResolvedValue({
        status: 200,
        ok: true,
        json: async () => mockFlag,
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      const result = await getFlag.run(ctx);

      expect(result.found).toBe(true);
      expect(result.flag).toEqual(mockFlag);
      expect(result.message).toContain('Successfully retrieved');
    });

    it('should handle 404 gracefully', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 404,
        ok: false,
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'nonexistent-flag' },
        connection: mockConnection,
      } as any;

      const result = await getFlag.run(ctx);

      expect(result.found).toBe(false);
      expect(result.message).toContain('not found');
    });

    it('should throw error on API error', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 500,
        ok: false,
        text: async () => 'Internal Server Error',
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await expect(getFlag.run(ctx)).rejects.toThrow(/500/);
    });

    it('should throw error on network failure', async () => {
      (global.fetch as any).mockRejectedValue(
        new Error('Network timeout'),
      );

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await expect(getFlag.run(ctx)).rejects.toThrow(/Request failed/i);
    });

    it('should URL-encode project and flag keys', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 200,
        ok: true,
        json: async () => ({
          key: 'test-flag',
          name: 'Test Flag',
          kind: 'boolean',
          _version: 1,
          creationDate: 1234567890,
          temporary: false,
          tags: [],
          variations: [],
          archived: false,
          _links: {},
        }),
      });

      const ctx = {
        props: { projectKey: 'my project', flagKey: 'flag with spaces' },
        connection: mockConnection,
      } as any;

      await getFlag.run(ctx);

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('my%20project'),
        expect.any(Object),
      );
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('flag%20with%20spaces'),
        expect.any(Object),
      );
    });
  });
});
