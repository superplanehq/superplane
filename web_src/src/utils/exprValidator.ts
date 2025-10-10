/* eslint-disable @typescript-eslint/no-explicit-any */
let isLoaded = false;
let loadPromise: Promise<void> | null = null;

const debug = (message: string, ...args: any[]) => {
  console.log(`[ExprValidator] ${message}`, ...args);
};

interface ValidationResult {
  valid?: boolean;
  error?: string;
}

async function loadWasm(): Promise<void> {
  if (isLoaded) {
    debug('WebAssembly already loaded');
    return;
  }

  if (loadPromise) {
    debug('WebAssembly loading in progress, waiting...');
    return loadPromise;
  }

  debug('Starting WebAssembly load process');

  loadPromise = (async () => {
    try {
      debug('Loading wasm_exec.js script');

    
      const wasmExecScript = document.createElement('script');
      wasmExecScript.src = '/wasm_exec.js';

      await new Promise<void>((resolve, reject) => {
        wasmExecScript.onload = () => {
          debug('wasm_exec.js loaded successfully');
          resolve();
        };
        wasmExecScript.onerror = (error) => {
          debug('Failed to load wasm_exec.js', error);
          reject(new Error('Failed to load wasm_exec.js'));
        };
        document.head.appendChild(wasmExecScript);
      });

      debug('Initializing Go WebAssembly');

    
      if (typeof (window as any).Go !== 'function') {
        throw new Error('Go constructor not available after loading wasm_exec.js');
      }

    
      const go = new (window as any).Go();
      debug('Go instance created', go);

      debug('Fetching WebAssembly module');

    
      const wasmResponse = await fetch('/expr-validator.wasm');
      if (!wasmResponse.ok) {
        throw new Error(`Failed to fetch WASM module: ${wasmResponse.status} ${wasmResponse.statusText}`);
      }

      const wasmBytes = await wasmResponse.arrayBuffer();
      debug('WebAssembly module fetched, size:', wasmBytes.byteLength, 'bytes');

      debug('Instantiating WebAssembly module');
      const wasmModule = await WebAssembly.instantiate(wasmBytes, go.importObject);
      debug('WebAssembly module instantiated');

    
      debug('Running Go program');
      go.run(wasmModule.instance);

    
      await new Promise(resolve => setTimeout(resolve, 100));

    
      if (typeof (window as any).validateBooleanExpression !== 'function') {
        throw new Error('validateBooleanExpression function not available after running Go program');
      }

      debug('WebAssembly module loaded and function is available');
      isLoaded = true;
    } catch (error) {
      debug('Failed to load WebAssembly module:', error);
      loadPromise = null;
      throw error;
    }
  })();

  return loadPromise;
}

/**
 * Validates a boolean expression using the Go expr-lang library via WebAssembly
 * @param expression - The boolean expression to validate
 * @param variables - Variables available in the expression context
 * @param filterType - Type of filter: 'data' or 'header'
 * @returns Promise<ValidationResult> - Validation result
 */
export async function validateBooleanExpression(
  expression: string,
  variables: Record<string, unknown> = {},
  filterType: 'data' | 'header' = 'data'
): Promise<ValidationResult> {
  try {
  
    await loadWasm();

  
    if (typeof (window as any).validateBooleanExpression !== 'function') {
      throw new Error('WebAssembly module not properly loaded');
    }

  
    const result = (window as any).validateBooleanExpression(
      expression,
      variables,
      filterType
    ) as ValidationResult;

    return result;
  } catch (error) {
    return {
      error: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
}

/**
 * Preload the WebAssembly module (optional, for performance)
 */
export async function preloadExprValidator(): Promise<void> {
  try {
    await loadWasm();
  } catch (error) {
    console.warn('Failed to preload expression validator:', error);
  }
}

export type { ValidationResult };
export const FilterTypes = {
  DATA: 'data' as const,
  HEADER: 'header' as const,
} as const;

export type FilterType = typeof FilterTypes[keyof typeof FilterTypes];