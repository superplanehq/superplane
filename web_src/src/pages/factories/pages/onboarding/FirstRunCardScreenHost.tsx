import { FirstRunCardForm } from "./first-run/FirstRunCardForm";
import { FirstRunCardScreen } from "./first-run/FirstRunCardScreen";
import { FirstRunCardReturnScreen } from "./first-run/FirstRunCardReturnScreen";
import type { FirstRunSphereProps } from "./first-run/FirstRunSpherePane";
import type { FirstRunChrome } from "./first-run/firstRunTypes";
import { useOnboardingCardFlow, type CardStepFlow } from "./useOnboardingCardFlow";

export function FirstRunCardScreenHost({
  organizationId,
  flow,
  chrome,
  sphere,
}: {
  organizationId: string;
  flow: CardStepFlow;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
}) {
  const { cardStep } = flow;
  const { session, returned, error, working, plan, continueSetup, cardSaved, selectPlan, backToPlans, retryCard } =
    useOnboardingCardFlow(organizationId, flow);
  if (session)
    return (
      <FirstRunCardForm
        key={session.token}
        session={session}
        chrome={chrome}
        sphere={sphere}
        onSaved={cardSaved}
        onBack={backToPlans}
        onRetry={retryCard}
      />
    );

  if (returned) {
    return (
      <FirstRunCardReturnScreen
        chrome={chrome}
        sphere={sphere}
        busy={working || flow.busy || !cardStep.ready}
        error={error}
        onRetry={() => void continueSetup(true)}
        onBack={backToPlans}
      />
    );
  }

  return (
    <FirstRunCardScreen
      plans={cardStep.plans}
      welcomeCreditCents={cardStep.welcomeCreditCents}
      selectedPlanKey={plan?.key}
      onSelectPlan={selectPlan}
      phase={
        working
          ? "starting"
          : cardStep.planActive
            ? "plan-active"
            : cardStep.hasPaymentMethod
              ? "card-saved"
              : "add-card"
      }
      error={error || undefined}
      busy={flow.busy || !cardStep.ready}
      chrome={chrome}
      sphere={sphere}
      onContinue={() => void continueSetup(false)}
    />
  );
}
