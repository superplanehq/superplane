import { factoryLineDetailPath } from "./factoryPagePaths";

const CUSTOM_VERIFICATION_PARAM = "customVerification";
const NEW_CUSTOM_VERIFICATION = "new";

export function factoryCustomVerificationSetupPath(
  organizationId: string,
  factoryKey: string,
  lineId: string,
  automationId?: string,
) {
  const params = new URLSearchParams({
    [CUSTOM_VERIFICATION_PARAM]: automationId?.trim() || NEW_CUSTOM_VERIFICATION,
  });
  return `${factoryLineDetailPath(organizationId, factoryKey, lineId)}?${params.toString()}`;
}

export function customVerificationAutomationIdFromSearch(search: string): string | null | undefined {
  const value = new URLSearchParams(search).get(CUSTOM_VERIFICATION_PARAM)?.trim();
  if (!value) {
    return undefined;
  }
  return value === NEW_CUSTOM_VERIFICATION ? null : value;
}
