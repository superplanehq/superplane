declare module 'react-element-to-jsx-string' {
  import { ReactNode } from 'react';

  type FilterPropsFunction = (value: any, key: string) => boolean

  interface ReactElementToJSXStringOptions {
    displayName?: (element: ReactNode) => string;
    filterProps?: string[] | FilterPropsFunction;
    showDefaultProps?: boolean;
    showFunctions?: boolean;
    functionValue?: (fn: any) => any;
    tabStop?: number;
    useBooleanShorthandSyntax?: boolean;
    maxInlineAttributesLineLength?: number;
    sortProps?: boolean;
    useFragmentShortSyntax?: boolean;
  }

  const reactElementToJSXString: (element: ReactNode, options?: ReactElementToJSXStringOptions) => string;

  export { ReactElementToJSXStringOptions as Options };

  export default reactElementToJSXString;
}
