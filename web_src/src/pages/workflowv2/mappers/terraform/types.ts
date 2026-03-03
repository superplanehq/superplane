export interface TerraformEventData {
  runId: string;
  workspaceId: string;
  action: string;
  runStatus: string;
  runUrl: string;
  runMessage: string;
  workspaceName: string;
  organizationName: string;
  runCreatedBy: string;
}
