export interface NodeMetadata {
  repository?: {
    uuid?: string;
    name?: string;
    full_name?: string;
    slug?: string;
  };
}

export interface BitbucketAccount {
  uuid?: string;
  account_id?: string;
  display_name?: string;
  nickname?: string;
}

export interface BitbucketLinks {
  self?: { href?: string };
  html?: { href?: string };
  commit?: { href?: string };
}

export interface BitbucketEndpoint {
  branch?: { name?: string };
  commit?: { hash?: string; links?: BitbucketLinks };
  repository?: { name?: string; full_name?: string };
}

export interface PullRequest {
  id?: number;
  title?: string;
  description?: string;
  state?: string;
  draft?: boolean;
  created_on?: string;
  updated_on?: string;
  close_source_branch?: boolean;
  comment_count?: number;
  task_count?: number;
  reason?: string;
  author?: BitbucketAccount;
  closed_by?: BitbucketAccount;
  reviewers?: BitbucketAccount[];
  source?: BitbucketEndpoint;
  destination?: BitbucketEndpoint;
  merge_commit?: { hash?: string };
  links?: BitbucketLinks;
}

export interface PullRequestComment {
  id?: number;
  created_on?: string;
  updated_on?: string;
  deleted?: boolean;
  content?: { raw?: string; markup?: string; html?: string };
  user?: BitbucketAccount;
  links?: BitbucketLinks;
}

export interface CommitStatus {
  key?: string;
  name?: string;
  state?: string;
  url?: string;
  description?: string;
  type?: string;
  refname?: string;
  created_on?: string;
  updated_on?: string;
  commit?: { hash?: string };
  links?: BitbucketLinks;
}

export interface CombinedCommitStatus {
  commit?: string;
  state?: string;
  total_count?: number;
  statuses?: CommitStatus[];
}

/** Payload of the bitbucket.onPullRequest trigger. */
export interface PullRequestEvent {
  actor?: BitbucketAccount;
  repository?: { name?: string; full_name?: string };
  pullrequest?: PullRequest;
}

/** Payload of the bitbucket.onPRComment trigger. */
export interface PullRequestCommentEvent extends PullRequestEvent {
  comment?: PullRequestComment;
}

/** Payload of the bitbucket.onCommitStatus trigger. */
export interface CommitStatusEvent {
  actor?: BitbucketAccount;
  repository?: { name?: string; full_name?: string };
  commit_status?: CommitStatus;
}
