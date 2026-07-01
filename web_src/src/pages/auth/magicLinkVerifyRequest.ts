interface MagicLinkVerifyRequestInput {
  token: string;
  inviteToken: string;
  redirectTarget: string;
  signupIntent: boolean;
}

export function buildMagicLinkVerifyRequest({
  token,
  inviteToken,
  redirectTarget,
  signupIntent,
}: MagicLinkVerifyRequestInput) {
  const formData = new URLSearchParams();
  formData.append("token", token);

  if (signupIntent) {
    formData.append("signup", "true");
  }

  if (inviteToken) {
    formData.append("invite_token", inviteToken);
  }

  const url = redirectTarget
    ? `/auth/magic-code/verify?redirect=${encodeURIComponent(redirectTarget)}`
    : "/auth/magic-code/verify";

  return {
    url,
    body: formData.toString(),
  };
}
