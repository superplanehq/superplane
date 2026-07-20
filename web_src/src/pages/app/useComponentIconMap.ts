import { useMemo } from "react";

interface IconSource {
  name?: string;
  icon?: string;
}

export function useComponentIconMap(components: IconSource[], triggers: IconSource[]) {
  return useMemo(() => {
    const map: Record<string, string> = {};

    for (const component of components) {
      if (component.name && component.icon) map[component.name] = component.icon;
    }

    for (const trigger of triggers) {
      if (trigger.name && trigger.icon) map[trigger.name] = trigger.icon;
    }

    return map;
  }, [components, triggers]);
}
