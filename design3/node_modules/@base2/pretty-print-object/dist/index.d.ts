interface PrettyPrintOptions {
    /**
     * Preferred indentation.
     *
     * @default '\t'
     */
    indent?: string;
    /**
     * Set to false to get double-quoted strings.
     *
     * @default true
     */
    singleQuotes?: boolean;
    /**
     * Whether to include the property prop of the object obj in the output.
     *
     * @param obj
     * @param prop
     */
    filter?: (obj: any, prop: string | symbol | number) => boolean;
    /**
     * Expected to return a string that transforms the string that resulted from stringifying obj[prop].
     * This can be used to detect special types of objects that need to be stringified in a particular way.
     * The transform function might return an alternate string in this case, otherwise returning the originalResult.
     *
     * @param obj
     * @param prop
     * @param originalResult
     */
    transform?: (obj: any, prop: string | symbol | number, originalResult: string) => string;
    /**
     * When set, will inline values up to inlineCharacterLimit length for the sake of more terse output.
     */
    inlineCharacterLimit?: number;
}
export declare function prettyPrint(input: any): string;
export declare function prettyPrint(input: any, options: PrettyPrintOptions): string;
export declare function prettyPrint(input: any, options: PrettyPrintOptions, pad: string): string;
export {};
