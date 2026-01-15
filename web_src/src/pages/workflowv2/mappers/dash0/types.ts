export interface QueryGraphQLConfiguration {
  query: string;
  variables?: Record<string, any>;
}

export interface QueryGraphQLMetadata {
  // No metadata needed initially
}

export interface GraphQLResponse {
  data?: Record<string, any>;
  errors?: Array<{
    message: string;
    locations?: Array<{ line: number; column: number }>;
    path?: any[];
  }>;
}
