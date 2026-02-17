import launchDarklyIntegration, {
  launchDarklyIntegration as namedIntegration,
  launchDarklyConnection,
  onFlagChange,
  getFlag,
  deleteFlag,
  LD_API_BASE_URL,
} from './index';

describe('index exports', () => {
  it('exports integration as both default and named export', () => {
    expect(launchDarklyIntegration).toBe(namedIntegration);
    expect(launchDarklyIntegration.name).toBe('LaunchDarkly');
  });

  it('wires connection, triggers, and actions', () => {
    expect(launchDarklyIntegration.connection).toBe(launchDarklyConnection);
    expect(launchDarklyIntegration.triggers).toEqual([onFlagChange]);
    expect(launchDarklyIntegration.actions).toEqual([getFlag, deleteFlag]);
  });

  it('re-exports auth constants', () => {
    expect(LD_API_BASE_URL).toBe('https://app.launchdarkly.com/api/v2');
  });
});
