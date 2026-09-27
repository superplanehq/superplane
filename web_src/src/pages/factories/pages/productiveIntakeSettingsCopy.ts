export const PRODUCTIVE_INTAKE_SETTINGS_COPY = {
  filtersLabel: "Filters",
  excludeKeyTasks: "Ignore key tasks",
  excludeKeyTasksHelper: "SuperPlane skips Productive key tasks (milestones).",
  filterByTaskList: "Task is in one of these task lists",
  filterByTaskListHelper:
    "SuperPlane creates a task when a Productive task is created in a selected list or is moved onto it.",
  taskListsLoading: "Loading task lists from the project",
  taskListsEmpty: "No active task lists found in the project.",
  taskListsError: "SuperPlane could not load the task lists. Try again later.",
  taskListsNoneSelected: "Select one or more task lists. With no selection, every task list creates a task.",
} as const;
