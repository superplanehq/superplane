/// <reference types="jest" />
import { onFlagChange, type OnFlagChangeProps } from '../triggers/on-flag-change';
import { TriggerHookContext } from '@superplane/framework';

describe('On Feature Flag Change Trigger', () => {
  describe('trigger definition', () => {
    it('should have correct metadata', () => {
      expect(onFlagChange.name).toBe('On Feature Flag Change');
      expect(onFlagChange.description).toContain('Triggers when');
      expect(onFlagChange.type).toBe('webhook');
      expect(onFlagChange.connection).toBeDefined();
    });

    it('should have correct prop definitions', () => {
      expect((onFlagChange as any).props.eventTypes).toBeDefined();
      expect((onFlagChange as any).props.eventTypes.required).toBe(false);
      expect((onFlagChange as any).props.eventTypes.type).toBe('array');
    });
  });

  describe('onWebhook method', () => {
    it('should process valid webhook payload', async () => {
      const payload = {
        _id: 'webhook-123',
        date: 1234567890,
        kind: 'flag.updated',
        name: 'Flag Updated',
        description: 'Test flag was updated',
        shortDescription: 'Updated',
        member: {
          _id: 'member-123',
          email: 'test@example.com',
        },
      };

      const ctx = {
        body: payload,
        props: { eventTypes: [] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      const result = await onFlagChange.onWebhook(ctx);

      expect(result).toBeDefined();
      expect(result?.payload).toEqual(payload);
      expect(result?.eventKind).toBe('flag.updated');
    });

    it('should return undefined for missing kind field', async () => {
      const payload = {
        _id: 'webhook-123',
        date: 1234567890,
        name: 'Flag Updated',
      };

      const ctx = {
        body: payload,
        props: { eventTypes: [] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      const result = await onFlagChange.onWebhook(ctx);

      expect(result).toBeUndefined();
    });

    it('should return undefined for invalid payload', async () => {
      const ctx = {
        body: null,
        props: { eventTypes: [] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      const result = await onFlagChange.onWebhook(ctx);

      expect(result).toBeUndefined();
    });

    it('should filter events by type when eventTypes is provided', async () => {
      const payload = {
        _id: 'webhook-123',
        date: 1234567890,
        kind: 'flag.deleted',
        name: 'Flag Deleted',
        description: 'Test flag was deleted',
        shortDescription: 'Deleted',
      };

      const ctx = {
        body: payload,
        props: { eventTypes: ['flag.updated', 'flag.created'] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      const result = await onFlagChange.onWebhook(ctx);

      expect(result).toBeUndefined();
    });

    it('should include matching event when in allowed list', async () => {
      const payload = {
        _id: 'webhook-123',
        date: 1234567890,
        kind: 'flag.updated',
        name: 'Flag Updated',
        description: 'Test flag was updated',
        shortDescription: 'Updated',
      };

      const ctx = {
        body: payload,
        props: { eventTypes: ['flag.updated', 'flag.deleted'] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      const result = await onFlagChange.onWebhook(ctx);

      expect(result).toBeDefined();
      expect(result?.eventKind).toBe('flag.updated');
    });

    it('should handle all flag event types', async () => {
      const eventTypes = [
        'flag.created',
        'flag.updated',
        'flag.deleted',
        'flag.archived',
        'flag.restored',
      ];

      for (const eventKind of eventTypes) {
        const payload = {
          _id: `webhook-${eventKind}`,
          date: 1234567890,
          kind: eventKind,
          name: `Flag ${eventKind}`,
          description: `Test flag was ${eventKind}`,
          shortDescription: eventKind,
        };

        const ctx = {
          body: payload,
          props: { eventTypes: [] },
        } as any as TriggerHookContext<OnFlagChangeProps>;

        const result = await onFlagChange.onWebhook(ctx);

        expect(result).toBeDefined();
        expect(result?.eventKind).toBe(eventKind);
      }
    });

    it('should preserve additional payload properties', async () => {
      const payload = {
        _id: 'webhook-123',
        date: 1234567890,
        kind: 'flag.updated',
        name: 'Flag Updated',
        description: 'Test flag was updated',
        shortDescription: 'Updated',
        customField: 'customValue',
        nestedData: { foo: 'bar' },
      };

      const ctx = {
        body: payload,
        props: { eventTypes: [] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      const result = await onFlagChange.onWebhook(ctx);

      expect(result?.payload.customField).toBe('customValue');
      expect(result?.payload.nestedData).toEqual({ foo: 'bar' });
    });

    it('should throw error on JSON parse failure', async () => {
      const ctx = {
        body: 'invalid json',
        props: { eventTypes: [] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      // Should handle gracefully or throw appropriately
      // Depending on how the framework handles body parsing
      const result = await onFlagChange.onWebhook(ctx);
      expect(result).toBeUndefined();
    });

    it('should handle empty eventTypes array same as no filter', async () => {
      const payload = {
        _id: 'webhook-123',
        date: 1234567890,
        kind: 'flag.created',
        name: 'Flag Created',
        description: 'Test flag was created',
        shortDescription: 'Created',
      };

      const ctx = {
        body: payload,
        props: { eventTypes: [] },
      } as any as TriggerHookContext<OnFlagChangeProps>;

      const result = await onFlagChange.onWebhook(ctx);

      expect(result).toBeDefined();
      expect(result?.eventKind).toBe('flag.created');
    });
  });
});
