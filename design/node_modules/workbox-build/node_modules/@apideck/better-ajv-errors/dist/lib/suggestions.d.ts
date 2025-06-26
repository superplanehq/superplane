export declare const getSuggestion: ({ value, suggestions, format, }: {
    value: string | null;
    suggestions: string[];
    format?: ((suggestion: string) => string) | undefined;
}) => string;
