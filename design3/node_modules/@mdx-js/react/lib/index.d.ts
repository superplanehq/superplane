/**
 * @param {import('react').ComponentType<any>} Component
 * @deprecated
 *   This export is marked as a legacy feature.
 *   That means it’s no longer recommended for use as it might be removed
 *   in a future major release.
 *
 *   Please use `useMDXComponents` to get context based components instead.
 */
export function withMDXComponents(Component: import('react').ComponentType<any>): (props: Record<string, unknown> & {
    components?: Components | null | undefined;
}) => JSX.Element;
/**
 * Get current components from the MDX Context.
 *
 * @param {Components | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that takes the current
 *   components and filters/merges/changes them.
 * @returns {Components}
 *   Current components.
 */
export function useMDXComponents(components?: Components | MergeComponents | null | undefined): Components;
/**
 * Provider for MDX context
 *
 * @param {Props} props
 * @returns {JSX.Element}
 */
export function MDXProvider({ components, children, disableParentContext }: Props): JSX.Element;
/**
 * @type {import('react').Context<Components>}
 * @deprecated
 *   This export is marked as a legacy feature.
 *   That means it’s no longer recommended for use as it might be removed
 *   in a future major release.
 *
 *   Please use `useMDXComponents` to get context based components and
 *   `MDXProvider` to set context based components instead.
 */
export const MDXContext: import('react').Context<Components>;
export type ReactNode = import('react').ReactNode;
export type Components = import('mdx/types.js').MDXComponents;
/**
 * Configuration.
 */
export type Props = {
    /**
     * Mapping of names for JSX components to React components.
     */
    components?: Components | MergeComponents | null | undefined;
    /**
     * Turn off outer component context.
     */
    disableParentContext?: boolean | null | undefined;
    /**
     * Children.
     */
    children?: ReactNode | null | undefined;
};
/**
 * Custom merge function.
 */
export type MergeComponents = (currentComponents: Components) => Components;
