export interface AppCardData {
  id: string;
  displayName: string;
  slug: string;
  description?: string;
  createdAt: string;
  syncStatus?: string;
  liveCommitSha?: string;
  canvasId?: string;
}
