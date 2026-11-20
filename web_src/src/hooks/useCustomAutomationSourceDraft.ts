import { useState } from "react";
import { getApiErrorMessage } from "@/lib/errors";
import type { CustomAutomationColumn } from "@/pages/factories/lib/customAutomation";
import {
  automationDefinitionYaml,
  parseAutomationForCanvas,
  type AutomationDefinition,
} from "@/pages/factories/lib/customAutomationDefinition";

export function useCustomAutomationSourceDraft(initial: string, stage: CustomAutomationColumn) {
  const [initialSource] = useState(initial);
  const [source, setSource] = useState(initialSource);
  const [tab, setTab] = useState<"canvas" | "yaml">("canvas");
  const [problem, setProblem] = useState("");
  const [initialCanvas] = useState(() => projectSource(initialSource, stage));
  const [definition, setDefinition] = useState(initialCanvas.definition);
  const editYaml = (yaml: string) => {
    setSource(yaml);
    setProblem("");
    setDefinition(undefined);
  };
  const editCanvas = (patch: Partial<AutomationDefinition>) => {
    if (!definition) return;
    const next = { ...definition, ...patch };
    const yaml = automationDefinitionYaml(next);
    if (yaml === automationDefinitionYaml(definition)) return;
    setDefinition(next);
    setSource(yaml);
  };
  const switchTab = (next: string) => {
    if (next === "yaml") {
      setTab("yaml");
      return;
    }
    if (!definition) {
      const projected = projectSource(source, stage);
      if (!projected.definition) {
        setProblem(projected.problem);
        return;
      }
      setDefinition(projected.definition);
    }
    setProblem("");
    setTab("canvas");
  };
  return {
    tab: definition ? tab : ("yaml" as const),
    source,
    definition,
    dirty: source !== initialSource,
    problem: problem || (source === initialSource ? initialCanvas.problem : ""),
    editYaml,
    editCanvas,
    switchTab,
  };
}

function projectSource(source: string, stage: CustomAutomationColumn) {
  try {
    return { definition: parseAutomationForCanvas(source, stage), problem: "" };
  } catch (error) {
    return {
      definition: undefined,
      problem: getApiErrorMessage(error, "SuperPlane could not show this YAML on Canvas."),
    };
  }
}
