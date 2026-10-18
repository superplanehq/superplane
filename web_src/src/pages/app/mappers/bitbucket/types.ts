export interface NodeMetadata {
  repository?: {
    uuid?: string;
    name?: string;
    full_name?: string;
    slug?: string;
  };
}

interface BitbucketLinks {
  html?: { href?: string };
}

interface BitbucketPullRequestEndpoint {
  branch?: { name?: string };
  commit?: { hash?: string };
}

export interface BitbucketPullRequest {
  id?: number;
  title?: string;
  state?: string;
  draft?: boolean;
  source?: BitbucketPullRequestEndpoint;
  destination?: BitbucketPullRequestEndpoint;
  links?: BitbucketLinks;
}

export interface BitbucketPullRequestComment {
  id?: number;
  links?: BitbucketLinks;
}
