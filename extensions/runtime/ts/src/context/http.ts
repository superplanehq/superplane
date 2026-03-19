export interface HTTPRequest {
  method: string;
  url: string;
  headers?: Record<string, string>;
  body?: string | Uint8Array;
}

export interface HTTPResponse {
  status: number;
  headers: Record<string, string>;
  body: Uint8Array;
}

export interface HTTPContext {
  do(request: HTTPRequest): Promise<HTTPResponse>;
}
