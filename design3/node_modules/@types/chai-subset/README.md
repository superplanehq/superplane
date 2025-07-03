# Installation
> `npm install --save @types/chai-subset`

# Summary
This package contains type definitions for chai-subset (https://github.com/debitoor/chai-subset).

# Details
Files were exported from https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/chai-subset.
## [index.d.ts](https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/chai-subset/index.d.ts)
````ts
/// <reference types="chai" />

declare global {
    namespace Chai {
        interface Assertion {
            containSubset(expected: any): Assertion;
        }
        interface Assert {
            containSubset(val: any, exp: any, msg?: string): void;
        }
    }
}

declare const chaiSubset: Chai.ChaiPlugin;
export = chaiSubset;

````

### Additional Details
 * Last updated: Wed, 05 Mar 2025 19:02:12 GMT
 * Dependencies: none
 * Peer dependencies: [@types/chai](https://npmjs.com/package/@types/chai)

# Credits
These definitions were written by [Sam Noedel](https://github.com/delta62), and [Andrew Brown](https://github.com/AGBrown).
