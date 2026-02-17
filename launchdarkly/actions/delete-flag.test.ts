/// <reference types="jest" />
/// <reference types="node" />
import { deleteFlag } from '../actions/delete-flag';
import type { LaunchDarklyConnectionProps } from '../auth';

describe('Delete Feature Flag Action', () => {
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
      expect(deleteFlag.name).toBe('Delete Feature Flag');
      expect(deleteFlag.description).toContain('Permanently delete');
      expect(deleteFlag.connection).toBeDefined();
    });

    it('should have correct prop definitions', () => {
      expect((deleteFlag as any).props.projectKey).toBeDefined();
      expect((deleteFlag as any).props.projectKey.required).toBe(true);
      expect((deleteFlag as any).props.flagKey).toBeDefined();
      expect((deleteFlag as any).props.flagKey.required).toBe(true);
    });
  });

  describe('run method', () => {
    it('should throw error when projectKey is empty', async () => {
      const ctx = {
        props: { projectKey: '', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await expect(deleteFlag.run(ctx)).rejects.toThrow(
        'projectKey" is required and cannot be empty',
      );
    });

    it('should throw error when flagKey is empty', async () => {
      const ctx = {
        props: { projectKey: 'test-project', flagKey: '' },
        connection: mockConnection,
      } as any;

      await expect(deleteFlag.run(ctx)).rejects.toThrow(
        'flagKey" is required and cannot be empty',
      );
    });

    it('should delete flag successfully on 204', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 204,
        ok: true,
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      const result = await deleteFlag.run(ctx);

      expect(result.deleted).toBe(true);
      expect(result.statusCode).toBe(204);
      expect(result.message).toContain('successfully deleted');
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

      const result = await deleteFlag.run(ctx);

      expect(result.deleted).toBe(false);
      expect(result.statusCode).toBe(404);
      expect(result.message).toContain('not found');
    });

    it('should throw error on 409 conflict', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 409,
        ok: false,
        text: async () => 'Flag is in use by active experiment',
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await expect(deleteFlag.run(ctx)).rejects.toThrow(/conflict/i);
    });

    it('should throw error on other API errors', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 500,
        ok: false,
        text: async () => 'Internal Server Error',
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await expect(deleteFlag.run(ctx)).rejects.toThrow(/500/);
    });

    it('should throw error on network failure', async () => {
      (global.fetch as any).mockRejectedValue(
        new Error('Network timeout'),
      );

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await expect(deleteFlag.run(ctx)).rejects.toThrow(/Request failed/i);
    });

    it('should handle success with unexpected status code', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 200,
        ok: true,
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      const result = await deleteFlag.run(ctx);

      expect(result.deleted).toBe(true);
      expect(result.statusCode).toBe(200);
      expect(result.message).toContain('status 200');
    });

    it('should URL-encode project and flag keys', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 204,
        ok: true,
      });

      const ctx = {
        props: { projectKey: 'my project', flagKey: 'flag with spaces' },
        connection: mockConnection,
      } as any;

      await deleteFlag.run(ctx);

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('my%20project'),
        expect.any(Object),
      );
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('flag%20with%20spaces'),
        expect.any(Object),
      );
    });

    it('should use DELETE HTTP method', async () => {
      (global.fetch as any).mockResolvedValue({
        status: 204,
        ok: true,
      });

      const ctx = {
        props: { projectKey: 'test-project', flagKey: 'test-flag' },
        connection: mockConnection,
      } as any;

      await deleteFlag.run(ctx);

      expect(global.fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ method: 'DELETE' }),
      );
    });
  });
});
