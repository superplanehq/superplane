import * as readline from "readline";

export interface JsonRpcRequest {
  jsonrpc: "2.0";
  id?: number;
  method: string;
  params?: any;
}

export interface JsonRpcResponse {
  jsonrpc: "2.0";
  id: number;
  result?: any;
  error?: JsonRpcError;
}

export interface JsonRpcError {
  code: number;
  message: string;
  data?: any;
}

type RequestHandler = (
  method: string,
  params: any
) => Promise<any>;

/**
 * Bidirectional JSON-RPC 2.0 transport over stdin/stdout.
 *
 * - Receives requests from the Go server on stdin
 * - Sends responses back on stdout
 * - Can also send requests TO the Go server and receive responses
 */
export class RpcTransport {
  private nextId = 1;
  private pendingRequests = new Map<
    number,
    { resolve: (value: any) => void; reject: (error: Error) => void }
  >();
  private requestHandler: RequestHandler;
  private rl: readline.Interface;

  constructor(requestHandler: RequestHandler) {
    this.requestHandler = requestHandler;

    this.rl = readline.createInterface({
      input: process.stdin,
      terminal: false,
    });

    this.rl.on("line", (line: string) => {
      this.handleLine(line);
    });

    this.rl.on("close", () => {
      process.exit(0);
    });
  }

  private async handleLine(line: string): Promise<void> {
    let msg: any;
    try {
      msg = JSON.parse(line);
    } catch {
      return;
    }

    // If it has a method, it's a request (from Go to us)
    if (msg.method) {
      await this.handleIncomingRequest(msg);
      return;
    }

    // If it has an id and result/error, it's a response to one of our requests
    if (msg.id !== undefined && (msg.result !== undefined || msg.error)) {
      this.handleIncomingResponse(msg as JsonRpcResponse);
      return;
    }
  }

  private async handleIncomingRequest(req: JsonRpcRequest): Promise<void> {
    try {
      const result = await this.requestHandler(req.method, req.params);
      if (req.id !== undefined) {
        this.sendMessage({
          jsonrpc: "2.0",
          id: req.id,
          result: result ?? null,
        });
      }
    } catch (err: any) {
      if (req.id !== undefined) {
        this.sendMessage({
          jsonrpc: "2.0",
          id: req.id,
          error: {
            code: -32000,
            message: err.message || String(err),
          },
        });
      }
    }
  }

  private handleIncomingResponse(resp: JsonRpcResponse): void {
    const pending = this.pendingRequests.get(resp.id);
    if (!pending) return;

    this.pendingRequests.delete(resp.id);

    if (resp.error) {
      pending.reject(new Error(resp.error.message));
    } else {
      pending.resolve(resp.result);
    }
  }

  /**
   * Send a request to the Go server and wait for a response.
   * Used for context callbacks (secrets, http, metadata, etc.).
   */
  async call(method: string, params?: any): Promise<any> {
    const id = this.nextId++;

    return new Promise<any>((resolve, reject) => {
      this.pendingRequests.set(id, { resolve, reject });

      this.sendMessage({
        jsonrpc: "2.0",
        id,
        method,
        params,
      });
    });
  }

  private sendMessage(msg: any): void {
    const line = JSON.stringify(msg);
    process.stdout.write(line + "\n");
  }
}
