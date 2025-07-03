# Installation
> `npm install --save @types/detect-port`

# Summary
This package contains type definitions for detect-port (https://github.com/node-modules/detect-port).

# Details
Files were exported from https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/detect-port.
## [index.d.ts](https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/detect-port/index.d.ts)
````ts
type DetectPortCallback = (err: Error, _port: number) => void;

interface PortConfig {
    port?: number;
    hostname?: string | undefined;
    callback?: DetectPortCallback | undefined;
}

interface DetectPort {
    (port: number | PortConfig | undefined, callback: DetectPortCallback): void;
    (port?: number | PortConfig): Promise<number>;
}
declare const detectPort: DetectPort;
export = detectPort;

````

### Additional Details
 * Last updated: Mon, 06 Nov 2023 22:41:05 GMT
 * Dependencies: none

# Credits
These definitions were written by [Ivan Medina](https://github.com/ivandevp).
