export interface BitbucketNodeMetadata {
  repository?: {
    uuid?: string;
    name?: string;
    full_name?: string;
    slug?: string;
    links?: {
      html?: {
        href?: string;
      };
    };
  };
}

export interface BitbucketPushConfiguration {
  repository?: string;
  refs: Array<{
    type: string;
    value: string;
  }>;
}

export interface BitbucketPush {
  actor?: {
    display_name?: string;
    uuid?: string;
    nickname?: string;
  };
  repository?: {
    full_name?: string;
    name?: string;
    uuid?: string;
    links?: {
      html?: {
        href?: string;
      };
    };
  };
  push?: {
    changes?: BitbucketChange[];
  };
}

export interface BitbucketChange {
  new?: {
    type?: string;
    name?: string;
    target?: {
      hash?: string;
      message?: string;
      date?: string;
      author?: {
        raw?: string;
        user?: {
          display_name?: string;
          uuid?: string;
        };
      };
      links?: {
        html?: {
          href?: string;
        };
      };
    };
  };
  old?: {
    type?: string;
    name?: string;
  };
  created?: boolean;
  forced?: boolean;
  closed?: boolean;
  commits?: BitbucketCommit[];
  truncated?: boolean;
}

export interface BitbucketCommit {
  hash?: string;
  message?: string;
  author?: {
    raw?: string;
  };
  links?: {
    html?: {
      href?: string;
    };
  };
}
