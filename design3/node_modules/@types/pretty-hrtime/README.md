# Installation
> `npm install --save @types/pretty-hrtime`

# Summary
This package contains type definitions for pretty-hrtime (https://github.com/robrich/pretty-hrtime).

# Details
Files were exported from https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/pretty-hrtime.
## [index.d.ts](https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/pretty-hrtime/index.d.ts)
````ts
export = prettyHrtime;

declare function prettyHrtime(hrTime: [number, number], options?: prettyHrtime.Options): string;

declare namespace prettyHrtime {
    interface Options {
        verbose?: boolean | undefined;
        precise?: boolean | undefined;
    }
}

````

### Additional Details
 * Last updated: Tue, 07 Nov 2023 09:09:39 GMT
 * Dependencies: none

# Credits
These definitions were written by [BendingBender](https://github.com/BendingBender).
