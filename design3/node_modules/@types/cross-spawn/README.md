# Installation
> `npm install --save @types/cross-spawn`

# Summary
This package contains type definitions for cross-spawn (https://github.com/moxystudio/node-cross-spawn).

# Details
Files were exported from https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/cross-spawn.
## [index.d.ts](https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/cross-spawn/index.d.ts)
````ts
/// <reference types="node" />

import * as child_process from "child_process";

declare namespace spawn {
    /**
     * The `spawn()` method spawns a new process using the given `command`, with
     * command line arguments in `args`. If omitted, `args` defaults to an empty array.
     */
    const spawn: typeof child_process.spawn;

    /**
     * The `spawn.sync()` method spawns a new process using the given `command`, with
     * command line arguments in `args`. If omitted, `args` defaults to an empty array.
     */
    const sync: typeof child_process.spawnSync;
}

/**
 * The `spawn()` method spawns a new process using the given `command`, with
 * command line arguments in `args`. If omitted, `args` defaults to an empty array.
 */
declare function spawn(command: string, options: child_process.SpawnOptions): child_process.ChildProcess;
declare function spawn(
    command: string,
    args?: readonly string[],
    options?: child_process.SpawnOptions,
): child_process.ChildProcess;

export = spawn;

````

### Additional Details
 * Last updated: Mon, 20 Nov 2023 23:36:24 GMT
 * Dependencies: [@types/node](https://npmjs.com/package/@types/node)

# Credits
These definitions were written by [Alorel](https://github.com/Alorel), and [ExE Boss](https://github.com/ExE-Boss).
