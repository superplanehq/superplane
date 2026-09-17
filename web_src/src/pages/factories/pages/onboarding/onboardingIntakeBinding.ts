import type { FactoriesFactoryIntake } from "@/api-client";

const STORAGE_PREFIX = "superplane:onboarding-intake-binding";

export type OnboardingIntakeBinding = {
  integrationId?: string;
  resourceId?: string;
};

type ListedIntakeBinding = FactoriesFactoryIntake & OnboardingIntakeBinding;

function storageKey(intakeId: string): string {
  return `${STORAGE_PREFIX}:${intakeId}`;
}

export function rememberOnboardingIntakeBinding(intakeId: string | undefined, binding: OnboardingIntakeBinding): void {
  if (!intakeId) return;
  sessionStorage.setItem(storageKey(intakeId), JSON.stringify(binding));
}

export function forgetOnboardingIntakeBinding(intakeId: string | undefined): void {
  if (!intakeId) return;
  sessionStorage.removeItem(storageKey(intakeId));
}

export function onboardingIntakeBinding(intake: FactoriesFactoryIntake): OnboardingIntakeBinding {
  const listed = intake as ListedIntakeBinding;
  if (listed.integrationId || listed.resourceId) {
    return { integrationId: listed.integrationId, resourceId: listed.resourceId };
  }
  return readRememberedBinding(intake.id);
}

function readRememberedBinding(intakeId: string | undefined): OnboardingIntakeBinding {
  if (!intakeId) return {};
  const raw = sessionStorage.getItem(storageKey(intakeId));
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw) as OnboardingIntakeBinding;
    return {
      integrationId: parsed.integrationId,
      resourceId: parsed.resourceId,
    };
  } catch {
    return {};
  }
}
