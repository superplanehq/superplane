import { useCallback, useState } from "react";

/** Keep the model choice through integration setup redirects. */
export function useOnboardingAgentModel(factoryId: string) {
  const key = `superplane:onboarding-agent-model:${factoryId}`;
  const [model, setModel] = useState(() => localStorage.getItem(key));
  const selectModel = useCallback(
    (value: string) => {
      localStorage.setItem(key, value);
      setModel(value);
    },
    [key],
  );
  return [model, selectModel] as const;
}
