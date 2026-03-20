import { getApiErrorMessage } from "@/utils/errors";

export interface UsageLimitNotice {
  title: string;
  description: string;
  href?: string;
  actionLabel?: string;
  toastMessage: string;
}

type UsageLimitCopy = Omit<UsageLimitNotice, "href" | "actionLabel"> & {
  actionLabel?: string;
  needsUsagePage?: boolean;
};

const usageLimitCopyByMessage: Record<string, UsageLimitCopy> = {
  "account organization limit exceeded": {
    title: "Organization limit reached",
    description:
      "This account already has the maximum number of organizations allowed by the current plan. Remove an unused organization or change the plan before creating another one.",
    toastMessage: "This account already has the maximum number of organizations allowed by the current plan.",
  },
  "organization canvas limit exceeded": {
    title: "Canvas limit reached",
    description:
      "This organization already has the maximum number of canvases allowed by the current plan. Remove an unused canvas or change the plan before creating another one.",
    toastMessage: "This organization already has the maximum number of canvases allowed by the current plan.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "canvas node limit exceeded": {
    title: "Canvas is too large for this plan",
    description:
      "This canvas has more nodes than the current plan allows. Reduce the number of nodes or change the plan before saving.",
    toastMessage: "This canvas has more nodes than the current plan allows.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization user limit exceeded": {
    title: "Member limit reached",
    description:
      "This organization already has the maximum number of members allowed by the current plan. Remove an inactive member or change the plan before adding another person.",
    toastMessage: "This organization already has the maximum number of members allowed by the current plan.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization integration limit exceeded": {
    title: "Integration limit reached",
    description:
      "This organization already has the maximum number of integrations allowed by the current plan. Remove an unused integration or change the plan before connecting another one.",
    toastMessage: "This organization already has the maximum number of integrations allowed by the current plan.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization usage limit exceeded": {
    title: "Usage limit reached",
    description:
      "This organization has reached a configured usage limit for the current plan. Review usage or change the plan before trying again.",
    toastMessage: "This organization has reached a configured usage limit for the current plan.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization exceeds configured account usage limits": {
    title: "Account usage limit reached",
    description:
      "This organization is blocked because its linked account is already over the configured limits in the usage service. Reduce usage on the linked account or update the account limits before trying again.",
    toastMessage:
      "This organization is blocked because its linked account is already over the configured usage limits.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization has no billing account candidate": {
    title: "Usage account is not configured",
    description:
      "SuperPlane could not determine which billing account should own usage for this organization, so it cannot verify limits yet.",
    toastMessage: "SuperPlane could not determine which billing account should own usage for this organization.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "failed to set up organization usage": {
    title: "Usage setup failed",
    description:
      "SuperPlane could not set up usage tracking for this organization, so it could not verify limits. Try again in a moment.",
    toastMessage: "SuperPlane could not set up usage tracking for this organization.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
};

export function getUsageLimitNotice(error: unknown, organizationId?: string): UsageLimitNotice | null {
  const message = getApiErrorMessage(error, "").trim().toLowerCase();
  if (!message) {
    return null;
  }

  const copy = usageLimitCopyByMessage[message];
  if (!copy) {
    return null;
  }

  return {
    ...copy,
    href: copy.needsUsagePage && organizationId ? `/${organizationId}/settings/billing` : undefined,
  };
}

export function getUsageLimitToastMessage(error: unknown, fallback: string): string {
  return getUsageLimitNotice(error)?.toastMessage || getApiErrorMessage(error, fallback);
}
