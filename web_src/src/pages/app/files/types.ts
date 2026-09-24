export type AppFile = {
  path: string;
  content: string;
  language?: string;
  loading?: boolean;
  errorMessage?: string;
};

export type PendingFileChange =
  | {
      type: "added";
      path: string;
      content: string;
    }
  | {
      type: "modified";
      path: string;
      content: string;
    }
  | {
      type: "deleted";
      path: string;
    };
