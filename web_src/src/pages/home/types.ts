export interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  isStarred?: boolean;
  starredAt?: string;
  createdBy: { name: string };
}
