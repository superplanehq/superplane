/// <reference types="react" />
import { compiler, RuleType } from './index.tsx';
declare const _default: import("react").FC<Omit<import("react").HTMLAttributes<Element>, "children"> & {
    children: string;
    options?: Partial<{
        createElement: (tag: string | import("react").FunctionComponent<{}> | import("react").ComponentClass<{}, any>, props: import("react").JSX.IntrinsicAttributes, ...children: import("react").ReactNode[]) => import("react").ReactNode;
        disableAutoLink: boolean;
        disableParsingRawHTML: boolean;
        enforceAtxHeadings: boolean;
        forceBlock: boolean;
        forceInline: boolean;
        forceWrapper: boolean;
        namedCodesToUnicode: {
            [key: string]: string;
        };
        overrides: import("./index.tsx").MarkdownToJSX.Overrides;
        renderRule: (next: () => import("react").ReactNode, node: import("./index.tsx").MarkdownToJSX.ParserResult, renderChildren: import("./index.tsx").MarkdownToJSX.RuleOutput, state: import("./index.tsx").MarkdownToJSX.State) => import("react").ReactNode;
        sanitizer: (value: string, tag: keyof import("react").JSX.IntrinsicElements, attribute: string) => string;
        slugify: (input: string, defaultFn: (input: string) => string) => string;
        wrapper: import("react").ElementType<any>;
    }>;
}> & {
    compiler: typeof compiler;
    RuleType: typeof RuleType;
};
export default _default;
