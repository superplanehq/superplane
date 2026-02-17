export interface Issue {
  id?: string;
  identifier?: string;
  title?: string;
  description?: string;
  priority?: number;
  url?: string;
  createdAt?: string;
  team?: { id?: string };
  state?: { id?: string };
  assignee?: { id?: string };
}
