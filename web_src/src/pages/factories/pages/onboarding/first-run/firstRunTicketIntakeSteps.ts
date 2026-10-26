import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunStepperStep } from "./FirstRunStepper";
import type { FirstRunTicketSource } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.tickets;

export type FirstRunTicketStep = "connect" | "project" | "settings";

export type TicketProject = { id?: string; name?: string };

export type FirstRunTicketProjectState = {
  jiraProjects: TicketProject[];
  jiraProjectsLoading: boolean;
  jiraProjectsError: boolean;
  jiraProjectId: string;
  linearProjects: TicketProject[];
  linearProjectsLoading: boolean;
  linearProjectsError: boolean;
  linearProjectIds: string[];
};

export function ticketSourceName(source: FirstRunTicketSource): string {
  if (source === "jira") return copy.jira;
  if (source === "linear") return copy.linear;
  return copy.githubIssues;
}

function projectName(projects: TicketProject[], id: string): string {
  return projects.find((project) => project.id === id)?.name ?? id;
}

function projectStep(
  source: FirstRunTicketSource,
  repository: string | undefined,
  projects: FirstRunTicketProjectState,
): FirstRunStepperStep<FirstRunTicketStep> {
  if (source === "jira") {
    return {
      id: "project",
      label: copy.jiraProjectHeading,
      doneLabel: projects.jiraProjectId
        ? copy.jiraProjectDone(projectName(projects.jiraProjects, projects.jiraProjectId))
        : undefined,
    };
  }
  if (source === "linear") {
    return {
      id: "project",
      label: copy.linearProjectHeading,
      doneLabel: projects.linearProjectIds.length
        ? copy.linearProjectsDone(projects.linearProjectIds.length)
        : undefined,
    };
  }
  return {
    id: "project",
    label: copy.stepRepository,
    doneLabel: repository ? copy.stepRepositoryDone(repository) : undefined,
  };
}

export function ticketIntakeSteps(
  source: FirstRunTicketSource,
  repository: string | undefined,
  projects: FirstRunTicketProjectState,
): FirstRunStepperStep<FirstRunTicketStep>[] {
  return [
    { id: "connect", label: copy.stepConnect(ticketSourceName(source)) },
    projectStep(source, repository, projects),
    { id: "settings", label: copy.stepSettings },
  ];
}

/** A project choice that the user can confirm. GitHub reuses the chosen repository. */
export function ticketProjectChoiceComplete(source: FirstRunTicketSource, projects: FirstRunTicketProjectState) {
  if (source === "jira") {
    return Boolean(projects.jiraProjectId) && !projects.jiraProjectsLoading && !projects.jiraProjectsError;
  }
  if (source === "linear") {
    return projects.linearProjectIds.length > 0 && !projects.linearProjectsLoading && !projects.linearProjectsError;
  }
  return true;
}

export function currentTicketStep(args: {
  source: FirstRunTicketSource;
  connected: boolean;
  projectsConfirmed: boolean;
}): FirstRunTicketStep {
  if (!args.connected) return "connect";
  if (args.source !== "github-issues" && !args.projectsConfirmed) return "project";
  return "settings";
}
