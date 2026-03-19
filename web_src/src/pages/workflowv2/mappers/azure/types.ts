export interface BlobEventData {
  api?: string;
  clientRequestId?: string;
  requestId?: string;
  eTag?: string;
  contentType?: string;
  contentLength?: number;
  blobType?: string;
  url?: string;
  sequencer?: string;
}

export interface AzureBlobEvent {
  id?: string;
  subject?: string;
  eventType?: string;
  eventTime?: string;
  data?: BlobEventData;
}

export interface ACRTarget {
  mediaType?: string;
  size?: number;
  digest?: string;
  length?: number;
  repository?: string;
  tag?: string;
  url?: string;
}

export interface ACRRequest {
  id?: string;
  addr?: string;
  host?: string;
  method?: string;
  useragent?: string;
}

export interface ACRActorInfo {
  name?: string;
}

export interface ACRSource {
  addr?: string;
  instanceID?: string;
}

export interface ACREventData {
  id?: string;
  timestamp?: string;
  action?: string;
  target?: ACRTarget;
  request?: ACRRequest;
  actor?: ACRActorInfo;
  source?: ACRSource;
}
