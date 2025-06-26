export interface ISuppression {
    file: string;
    scopeId: string;
    rule: string;
}
export interface IBulkSuppressionsConfig {
    serializedSuppressions: Set<string>;
    jsonObject: IBulkSuppressionsJson;
    newSerializedSuppressions: Set<string>;
    newJsonObject: IBulkSuppressionsJson;
}
export interface IBulkSuppressionsJson {
    suppressions: ISuppression[];
}
export declare function getSuppressionsConfigForEslintrcFolderPath(eslintrcFolderPath: string): IBulkSuppressionsConfig;
export declare function getAllBulkSuppressionsConfigsByEslintrcFolderPath(): [string, IBulkSuppressionsConfig][];
export declare function writeSuppressionsJsonToFile(eslintrcFolderPath: string, suppressionsConfig: IBulkSuppressionsConfig): void;
export declare function deleteBulkSuppressionsFileInEslintrcFolder(eslintrcFolderPath: string): void;
export declare function serializeSuppression({ file, scopeId, rule }: ISuppression): string;
//# sourceMappingURL=bulk-suppressions-file.d.ts.map