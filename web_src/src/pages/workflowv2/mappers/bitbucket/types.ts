export interface NodeMetadata {
  repository?: {
    uuid?: string;
    name?: string;
    full_name?: string;
    slug?: string;
  };
}

export interface Issue {
  id?: number;
  title?: string;
  state?: string;
  created_on?: string;
  updated_on?: string;
  reporter?: {
    display_name?: string;
  };
  assignee?: {
    display_name?: string;
  };
  links?: {
    html?: {
      href?: string;
    };
  };
}

export interface Comment {
  id?: number;
  created_on?: string;
  content?: {
    raw?: string;
  };
  user?: {
    display_name?: string;
  };
  links?: {
    html?: {
      href?: string;
    };
  };
}
