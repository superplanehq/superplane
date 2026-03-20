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
      "This account has reached its organization limit. Remove an unused organization or change the plan before creating another one.",
    toastMessage: "Organization limit reached for this account.",
  },
  "organization canvas limit exceeded": {
    title: "Canvas limit reached",
    description:
      "This organization has reached its canvas limit. Remove an unused canvas or change the plan before creating another one.",
    toastMessage: "Canvas limit reached for this organization.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "canvas node limit exceeded": {
    title: "Canvas is too large for this plan",
    description:
      "This canvas exceeds the node limit for the current plan. Reduce the number of nodes or change the plan before saving.",
    toastMessage: "Canvas node limit reached for this organization.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization user limit exceeded": {
    title: "Member limit reached",
    description:
      "This organization has reached its member limit. Remove an inactive member or change the plan before adding another person.",
    toastMessage: "Member limit reached for this organization.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization integration limit exceeded": {
    title: "Integration limit reached",
    description:
      "This organization has reached its integration limit. Remove an unused integration or change the plan before connecting another one.",
    toastMessage: "Integration limit reached for this organization.",
    needsUsagePage: true,
    actionLabel: "View usage",
  },
  "organization exceeds configured account usage limits": {
    title: "Account usage limit reached",
    description:
      "This organization cannot be synced because the linked account is already over its configured limits in the usage service.",
    toastMessage: "Account usage limits are blocking this organization.",
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
