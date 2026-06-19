export type SignupWaitlistConfig = {
  portalID: string;
  formID: string;
  region?: string;
};

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION?: string;
};

export const getSignupWaitlistConfig = (): SignupWaitlistConfig | null => {
  const win = window as SignupWaitlistWindow;
  const portalID = win.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID?.trim();
  const formID = win.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID?.trim();
  const region = win.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION?.trim();

  if (!portalID || !formID) {
    return null;
  }

  return { portalID, formID, region: region || undefined };
};

export const hasSignupWaitlistConfig = () => getSignupWaitlistConfig() !== null;
