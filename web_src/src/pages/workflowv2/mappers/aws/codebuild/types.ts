export interface CodeBuildProject {
  projectName?: string;
  projectArn?: string;
}

export interface CodeBuildConfiguration {
  region?: string;
  project?: string;
  sourceVersion?: string;
}

export interface CodeBuildTriggerConfiguration extends CodeBuildConfiguration {}

export interface CodeBuildTriggerMetadata {
  region?: string;
  subscriptionId?: string;
  project?: CodeBuildProject;
}

export interface CodeBuildBuildEventDetail {
  "project-name"?: string;
  "build-status"?: string;
  "build-id"?: string;
  "current-phase"?: string;
  "current-phase-context"?: string;
  version?: string;
  "additional-information"?: {
    initiator?: string;
    "source-version"?: string;
    logs?: {
      "deep-link"?: string;
    };
  };
}

export interface CodeBuildBuildEvent {
  account?: string;
  region?: string;
  time?: string;
  "detail-type"?: string;
  detail?: CodeBuildBuildEventDetail;
}

export interface CodeBuildBuildLogs {
  deepLink?: string;
  cloudWatchLogsArn?: string;
  groupName?: string;
  streamName?: string;
  status?: string;
}

export interface CodeBuildBuildOutput {
  id?: string;
  arn?: string;
  buildNumber?: number;
  currentPhase?: string;
  buildStatus?: string;
  projectName?: string;
  sourceVersion?: string;
  initiator?: string;
  startTime?: string;
  endTime?: string;
  logs?: CodeBuildBuildLogs;
}
