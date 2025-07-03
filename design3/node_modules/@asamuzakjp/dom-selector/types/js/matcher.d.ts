export class Matcher {
    constructor(selector: string, node: object, opt?: {
        warn?: boolean;
    });
    _onError(e: Error): void;
    _setup(node: object): Array<object>;
    _sortLeaves(leaves: Array<object>): Array<object>;
    _correspond(selector: string): Array<Array<object | undefined>>;
    _traverse(node?: object, walker?: object): object | null;
    _collectNthChild(anb: {
        a: number;
        b: number;
        reverse?: boolean;
        selector?: object;
    }, node: object): Set<object>;
    _collectNthOfType(anb: {
        a: number;
        b: number;
        reverse?: boolean;
    }, node: object): Set<object>;
    _matchAnPlusB(ast: object, node: object, nthName: string): Set<object>;
    _matchPseudoElementSelector(astName: string, opt?: {
        forgive?: boolean;
    }): void;
    _matchDirectionPseudoClass(ast: object, node: object): object | null;
    _matchLanguagePseudoClass(ast: object, node: object): object | null;
    _matchHasPseudoFunc(leaves: Array<object>, node: object): boolean;
    _matchLogicalPseudoFunc(astData: object, node: object): object | null;
    _matchPseudoClassSelector(ast: object, node: object, opt?: {
        forgive?: boolean;
    }): Set<object>;
    _matchAttributeSelector(ast: object, node: object): object | null;
    _matchClassSelector(ast: object, node: object): object | null;
    _matchIDSelector(ast: object, node: object): object | null;
    _matchTypeSelector(ast: object, node: object, opt?: {
        forgive?: boolean;
    }): object | null;
    _matchShadowHostPseudoClass(ast: object, node: object): object | null;
    _matchSelector(ast: object, node: object, opt?: object): Set<object>;
    _matchLeaves(leaves: Array<object>, node: object, opt?: object): boolean;
    _findDescendantNodes(leaves: Array<object>, baseNode: object): object;
    _matchCombinator(twig: object, node: object, opt?: {
        dir?: string;
        forgive?: boolean;
    }): Set<object>;
    _findNode(leaves: Array<object>, opt?: {
        node?: object;
        tree?: object;
    }): object | null;
    _findEntryNodes(twig: object, targetType: string): object;
    _getEntryTwig(branch: Array<object>, targetType: string): object;
    _collectNodes(targetType: string): Array<Array<object | undefined>>;
    _sortNodes(nodes: Array<object> | Set<object>): Array<object | undefined>;
    _matchNodes(targetType: string): Set<object>;
    _find(targetType: string): Set<object>;
    matches(): boolean;
    closest(): object | null;
    querySelector(): object | null;
    querySelectorAll(): Array<object | undefined>;
    #private;
}
