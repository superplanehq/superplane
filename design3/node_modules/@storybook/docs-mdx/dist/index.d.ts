import * as t from '@babel/types';

declare const extractImports: (root: t.File) => Record<string, string>;
declare const plugin: (store: any) => (root: any) => any;
declare const analyze: (code: string) => {
    title: any;
    of: any;
    name: any;
    isTemplate: any;
    imports: any;
};

export { analyze, extractImports, plugin };
