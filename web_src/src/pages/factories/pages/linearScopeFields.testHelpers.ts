import { screen } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";

import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

/** Opens the Linear project search and picks the project with this name. */
export async function selectLinearProject(user: UserEvent, name: string) {
  await user.click(screen.getByRole("combobox", { name: LINEAR_INTAKE_SETUP_COPY.wizardSearchProjects }));
  await user.click(await screen.findByRole("option", { name }));
}

/** Opens the Linear label search and picks the label with this name. */
export async function selectLinearLabel(user: UserEvent, name: string) {
  await user.click(screen.getByRole("combobox", { name: LINEAR_INTAKE_SETUP_COPY.labelSearch }));
  await user.click(await screen.findByRole("option", { name }));
}

/** The remove control of a selected project or label chip. */
export function linearSelectedChip(name: string) {
  return screen.getByRole("button", { name: `Remove ${name}` });
}

export function queryLinearSelectedChip(name: string) {
  return screen.queryByRole("button", { name: `Remove ${name}` });
}
