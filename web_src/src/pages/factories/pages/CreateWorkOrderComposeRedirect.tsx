import { WorkOrdersPage } from "./WorkOrdersPage";

/** Deep link for `/tasks/new`: show the list while the layout opens the create dialog. */
export function CreateWorkOrderComposeRedirect() {
  return <WorkOrdersPage />;
}
