export type SignupWaitlistConfig = {
  accountID: string;
  formID: string;
};

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_FORM_ID?: string;
};

export const getSignupWaitlistConfig = (): SignupWaitlistConfig | null => {
  const win = window as SignupWaitlistWindow;
  const accountID = win.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID?.trim();
  const formID = win.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_FORM_ID?.trim();

  if (!accountID || !formID) {
    return null;
  }

  return { accountID, formID };
};

export const hasSignupWaitlistConfig = () => getSignupWaitlistConfig() !== null;
