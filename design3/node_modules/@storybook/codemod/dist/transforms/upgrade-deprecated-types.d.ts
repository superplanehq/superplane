import { FileInfo, API } from 'jscodeshift';
import { BabelFile } from '@babel/core';

declare function transform(info: FileInfo, api: API, options: {
    parser?: string;
}): string;
declare const parser = "tsx";
declare function upgradeDeprecatedTypes(file: BabelFile): void;

export { transform as default, parser, upgradeDeprecatedTypes };
