import { FileInfo } from 'jscodeshift';

declare function jscodeshift(info: FileInfo): string;
declare function transform(source: string, baseName: string): [mdx: string, csf: string];
declare function nameToValidExport(name: string): string;

export { jscodeshift as default, nameToValidExport, transform };
