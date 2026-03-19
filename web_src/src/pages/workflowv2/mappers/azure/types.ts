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
