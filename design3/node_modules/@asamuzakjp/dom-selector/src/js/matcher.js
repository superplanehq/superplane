/**
 * matcher.js
 */

/* import */
import isCustomElementName from 'is-potential-custom-element-name';
import {
  getDirectionality, isContentEditable, isInclusive, isInShadowTree,
  isNamespaceDeclared, isPreceding, selectorToNodeProps
} from './dom-util.js';
import {
  generateCSS, parseSelector, unescapeSelector, walkAST
} from './parser.js';

/* constants */
import {
  ALPHA_NUM, BIT_01, BIT_02, BIT_04, BIT_08, BIT_16, BIT_32, COMBINATOR,
  DOCUMENT_FRAGMENT_NODE, DOCUMENT_NODE, ELEMENT_NODE, NOT_SUPPORTED_ERR,
  REG_LOGICAL_PSEUDO, REG_SHADOW_HOST, SELECTOR_ATTR, SELECTOR_CLASS,
  SELECTOR_ID, SELECTOR_PSEUDO_CLASS, SELECTOR_PSEUDO_ELEMENT, SELECTOR_TYPE,
  SHOW_ALL, SHOW_DOCUMENT, SHOW_DOCUMENT_FRAGMENT, SHOW_ELEMENT, SYNTAX_ERR,
  TEXT_NODE, TYPE_FROM, TYPE_TO
} from './constant.js';
const DIR_NEXT = 'next';
const DIR_PREV = 'prev';
const TARGET_ALL = 'all';
const TARGET_FIRST = 'first';
const TARGET_LINEAL = 'lineal';
const TARGET_SELF = 'self';

/**
 * Matcher
 * NOTE: #ast[i] corresponds to #nodes[i]
 * #ast: [
 *   {
 *     branch: branch[],
 *     dir: string|null,
 *     filtered: boolean,
 *     find: boolean
 *   },
 *   {
 *     branch: branch[],
 *     dir: string|null,
 *     filtered: boolean,
 *     find: boolean
 *   }
 * ]
 * #nodes: [
 *   Set([node{}, node{}]),
 *   Set([node{}, node{}, node{}])
 * ]
 * branch[]: [twig{}, twig{}]
 * twig{}: {
 *   combo: leaf{}|null,
 *   leaves: leaves[]
 * }
 * leaves[]: [leaf{}, leaf{}, leaf{}]
 * leaf{}: CSSTree AST object
 * node{}: Element node
 */
export class Matcher {
  /* private fields */
  #ast;
  #bit;
  #cache;
  #document;
  #finder;
  #node;
  #nodes;
  #root;
  #shadow;
  #tree;
  #warn;
  #window;

  /**
   * construct
   * @param {string} selector - CSS selector
   * @param {object} node - Document, DocumentFragment, Element node
   * @param {object} [opt] - options
   * @param {boolean} [opt.warn] - console warn
   */
  constructor(selector, node, opt = {}) {
    const { warn } = opt;
    this.#bit = new Map([
      [SELECTOR_PSEUDO_ELEMENT, BIT_01],
      [SELECTOR_ID, BIT_02],
      [SELECTOR_CLASS, BIT_04],
      [SELECTOR_TYPE, BIT_08],
      [SELECTOR_ATTR, BIT_16],
      [SELECTOR_PSEUDO_CLASS, BIT_32]
    ]);
    this.#cache = new WeakMap();
    this.#node = node;
    [this.#window, this.#document, this.#root, this.#tree] = this._setup(node);
    this.#shadow = isInShadowTree(node);
    [this.#ast, this.#nodes] = this._correspond(selector);
    this.#warn = !!warn;
  }

  /**
   * handle error
   * @param {Error} e - Error
   * @throws Error
   * @returns {void}
   */
  _onError(e) {
    if ((e instanceof DOMException ||
         e instanceof this.#window.DOMException) &&
        e.name === NOT_SUPPORTED_ERR) {
      if (this.#warn) {
        console.warn(e.message);
      }
    } else if (e instanceof DOMException) {
      throw new this.#window.DOMException(e.message, e.name);
    } else if (e instanceof TypeError) {
      throw new this.#window.TypeError(e.message);
    } else {
      throw e;
    }
  }

  /**
   * set up #window, #document, #root, #walker
   * @param {object} node - Document, DocumentFragment, Element node
   * @returns {Array.<object>} - array of #window, #document, #root, #walker
   */
  _setup(node) {
    let document;
    let root;
    switch (node?.nodeType) {
      case DOCUMENT_NODE: {
        document = node;
        root = node;
        break;
      }
      case DOCUMENT_FRAGMENT_NODE: {
        document = node.ownerDocument;
        root = node;
        break;
      }
      case ELEMENT_NODE: {
        if (node.ownerDocument.contains(node)) {
          document = node.ownerDocument;
          root = node.ownerDocument;
        } else {
          let parent = node;
          while (parent) {
            if (parent.parentNode) {
              parent = parent.parentNode;
            } else {
              break;
            }
          }
          document = parent.ownerDocument;
          root = parent;
        }
        break;
      }
      default: {
        let msg;
        if (node?.nodeName) {
          msg = `Unexpected node ${node.nodeName}`;
        } else {
          const nodeType =
            Object.prototype.toString.call(node).slice(TYPE_FROM, TYPE_TO);
          msg = `Unexpected node ${nodeType}`;
        }
        throw new TypeError(msg);
      }
    }
    const filter = SHOW_DOCUMENT | SHOW_DOCUMENT_FRAGMENT | SHOW_ELEMENT;
    const walker = document.createTreeWalker(root, filter);
    const window = document.defaultView;
    return [
      window,
      document,
      root,
      walker
    ];
  }

  /**
   * sort AST leaves
   * @param {Array.<object>} leaves - collection of AST leaves
   * @returns {Array.<object>} - sorted leaves
   */
  _sortLeaves(leaves) {
    const arr = [...leaves];
    if (arr.length > 1) {
      arr.sort((a, b) => {
        const { type: typeA } = a;
        const { type: typeB } = b;
        const bitA = this.#bit.get(typeA);
        const bitB = this.#bit.get(typeB);
        let res;
        if (bitA === bitB) {
          res = 0;
        } else if (bitA > bitB) {
          res = 1;
        } else {
          res = -1;
        }
        return res;
      });
    }
    return arr;
  }

  /**
   * correspond #ast and #nodes
   * @param {string} selector - CSS selector
   * @returns {Array.<Array.<object|undefined>>} - array of #ast and #nodes
   */
  _correspond(selector) {
    let cssAst;
    try {
      cssAst = parseSelector(selector);
    } catch (e) {
      this._onError(e);
    }
    const branches = walkAST(cssAst);
    const ast = [];
    const nodes = [];
    let i = 0;
    for (const [...items] of branches) {
      const branch = [];
      let item = items.shift();
      if (item && item.type !== COMBINATOR) {
        const leaves = new Set();
        while (item) {
          if (item.type === COMBINATOR) {
            const [nextItem] = items;
            if (nextItem.type === COMBINATOR) {
              const msg = `Invalid combinator ${item.name}${nextItem.name}`;
              throw new this.#window.DOMException(msg, SYNTAX_ERR);
            }
            branch.push({
              combo: item,
              leaves: this._sortLeaves(leaves)
            });
            leaves.clear();
          } else if (item) {
            leaves.add(item);
          }
          if (items.length) {
            item = items.shift();
          } else {
            branch.push({
              combo: null,
              leaves: this._sortLeaves(leaves)
            });
            leaves.clear();
            break;
          }
        }
      }
      ast.push({
        branch,
        dir: null,
        filtered: false,
        find: false
      });
      nodes[i] = new Set();
      i++;
    }
    return [
      ast,
      nodes
    ];
  }

  /**
   * traverse tree walker
   * @param {object} [node] - Element node
   * @param {object} [walker] - tree walker
   * @returns {?object} - current node
   */
  _traverse(node = {}, walker = this.#tree) {
    let current;
    let refNode = walker.currentNode;
    if (node.nodeType === ELEMENT_NODE && refNode === node) {
      current = refNode;
    } else {
      if (refNode !== walker.root) {
        while (refNode) {
          if (refNode === walker.root ||
              (node.nodeType === ELEMENT_NODE && refNode === node)) {
            break;
          }
          refNode = walker.parentNode();
        }
      }
      if (node.nodeType === ELEMENT_NODE) {
        while (refNode) {
          if (refNode === node) {
            current = refNode;
            break;
          }
          refNode = walker.nextNode();
        }
      } else {
        current = refNode;
      }
    }
    return current ?? null;
  }

  /**
   * collect nth child
   * @param {object} anb - An+B options
   * @param {number} anb.a - a
   * @param {number} anb.b - b
   * @param {boolean} [anb.reverse] - reverse order
   * @param {object} [anb.selector] - AST
   * @param {object} node - Element node
   * @returns {Set.<object>} - collection of matched nodes
   */
  _collectNthChild(anb, node) {
    const { a, b, reverse, selector } = anb;
    const { parentNode } = node;
    let matched = new Set();
    let selectorBranches;
    if (selector) {
      if (this.#cache.has(selector)) {
        selectorBranches = this.#cache.get(selector);
      } else {
        selectorBranches = walkAST(selector);
        this.#cache.set(selector, selectorBranches);
      }
    }
    if (parentNode) {
      const filter = SHOW_DOCUMENT | SHOW_DOCUMENT_FRAGMENT | SHOW_ELEMENT;
      const walker = this.#document.createTreeWalker(parentNode, filter);
      let l = 0;
      let refNode = walker.firstChild();
      while (refNode) {
        l++;
        refNode = walker.nextSibling();
      }
      refNode = this._traverse(parentNode, walker);
      const selectorNodes = new Set();
      if (selectorBranches) {
        refNode = this._traverse(parentNode, walker);
        refNode = walker.firstChild();
        while (refNode) {
          let bool;
          for (const leaves of selectorBranches) {
            bool = this._matchLeaves(leaves, refNode);
            if (!bool) {
              break;
            }
          }
          if (bool) {
            selectorNodes.add(refNode);
          }
          refNode = walker.nextSibling();
        }
      }
      // :first-child, :last-child, :nth-child(b of S), :nth-last-child(b of S)
      if (a === 0) {
        if (b > 0 && b <= l) {
          if (selectorNodes.size) {
            let i = 0;
            refNode = this._traverse(parentNode, walker);
            if (reverse) {
              refNode = walker.lastChild();
            } else {
              refNode = walker.firstChild();
            }
            while (refNode) {
              if (selectorNodes.has(refNode)) {
                if (i === b - 1) {
                  matched.add(refNode);
                  break;
                }
                i++;
              }
              if (reverse) {
                refNode = walker.previousSibling();
              } else {
                refNode = walker.nextSibling();
              }
            }
          } else if (!selector) {
            let i = 0;
            refNode = this._traverse(parentNode, walker);
            if (reverse) {
              refNode = walker.lastChild();
            } else {
              refNode = walker.firstChild();
            }
            while (refNode) {
              if (i === b - 1) {
                matched.add(refNode);
                break;
              }
              if (reverse) {
                refNode = walker.previousSibling();
              } else {
                refNode = walker.nextSibling();
              }
              i++;
            }
          }
        }
      // :nth-child()
      } else {
        let nth = b - 1;
        if (a > 0) {
          while (nth < 0) {
            nth += a;
          }
        }
        if (nth >= 0 && nth < l) {
          let i = 0;
          let j = a > 0 ? 0 : b - 1;
          refNode = this._traverse(parentNode, walker);
          if (reverse) {
            refNode = walker.lastChild();
          } else {
            refNode = walker.firstChild();
          }
          while (refNode) {
            if (refNode && nth >= 0 && nth < l) {
              if (selectorNodes.size) {
                if (selectorNodes.has(refNode)) {
                  if (j === nth) {
                    matched.add(refNode);
                    nth += a;
                  }
                  if (a > 0) {
                    j++;
                  } else {
                    j--;
                  }
                }
              } else if (i === nth) {
                if (!selector) {
                  matched.add(refNode);
                }
                nth += a;
              }
              if (reverse) {
                refNode = walker.previousSibling();
              } else {
                refNode = walker.nextSibling();
              }
              i++;
            } else {
              break;
            }
          }
        }
      }
      if (reverse && matched.size > 1) {
        const m = [...matched];
        matched = new Set(m.reverse());
      }
    } else if (node === this.#root && this.#root.nodeType === ELEMENT_NODE &&
               (a + b) === 1) {
      if (selectorBranches) {
        let bool;
        for (const leaves of selectorBranches) {
          bool = this._matchLeaves(leaves, node);
          if (bool) {
            break;
          }
        }
        if (bool) {
          matched.add(node);
        }
      } else {
        matched.add(node);
      }
    }
    return matched;
  }

  /**
   * collect nth of type
   * @param {object} anb - An+B options
   * @param {number} anb.a - a
   * @param {number} anb.b - b
   * @param {boolean} [anb.reverse] - reverse order
   * @param {object} node - Element node
   * @returns {Set.<object>} - collection of matched nodes
   */
  _collectNthOfType(anb, node) {
    const { a, b, reverse } = anb;
    const { localName, parentNode, prefix } = node;
    let matched = new Set();
    if (parentNode) {
      const filter = SHOW_DOCUMENT | SHOW_DOCUMENT_FRAGMENT | SHOW_ELEMENT;
      const walker = this.#document.createTreeWalker(parentNode, filter);
      let l = 0;
      let refNode = walker.firstChild();
      while (refNode) {
        l++;
        refNode = walker.nextSibling();
      }
      // :first-of-type, :last-of-type
      if (a === 0) {
        if (b > 0 && b <= l) {
          let j = 0;
          refNode = this._traverse(parentNode, walker);
          if (reverse) {
            refNode = walker.lastChild();
          } else {
            refNode = walker.firstChild();
          }
          while (refNode) {
            const { localName: itemLocalName, prefix: itemPrefix } = refNode;
            if (itemLocalName === localName && itemPrefix === prefix) {
              if (j === b - 1) {
                matched.add(refNode);
                break;
              }
              j++;
            }
            if (reverse) {
              refNode = walker.previousSibling();
            } else {
              refNode = walker.nextSibling();
            }
          }
        }
      // :nth-of-type()
      } else {
        let nth = b - 1;
        if (a > 0) {
          while (nth < 0) {
            nth += a;
          }
        }
        if (nth >= 0 && nth < l) {
          let j = a > 0 ? 0 : b - 1;
          refNode = this._traverse(parentNode, walker);
          if (reverse) {
            refNode = walker.lastChild();
          } else {
            refNode = walker.firstChild();
          }
          while (refNode) {
            const { localName: itemLocalName, prefix: itemPrefix } = refNode;
            if (itemLocalName === localName && itemPrefix === prefix) {
              if (j === nth) {
                matched.add(refNode);
                nth += a;
              }
              if (nth < 0 || nth >= l) {
                break;
              } else if (a > 0) {
                j++;
              } else {
                j--;
              }
            }
            if (reverse) {
              refNode = walker.previousSibling();
            } else {
              refNode = walker.nextSibling();
            }
          }
        }
      }
      if (reverse && matched.size > 1) {
        const m = [...matched];
        matched = new Set(m.reverse());
      }
    } else if (node === this.#root && this.#root.nodeType === ELEMENT_NODE &&
               (a + b) === 1) {
      matched.add(node);
    }
    return matched;
  }

  /**
   * match An+B
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @param {string} nthName - nth pseudo-class name
   * @returns {Set.<object>} - collection of matched nodes
   */
  _matchAnPlusB(ast, node, nthName) {
    const {
      nth: {
        a,
        b,
        name: nthIdentName
      },
      selector
    } = ast;
    const identName = unescapeSelector(nthIdentName);
    const anbMap = new Map();
    if (identName) {
      if (identName === 'even') {
        anbMap.set('a', 2);
        anbMap.set('b', 0);
      } else if (identName === 'odd') {
        anbMap.set('a', 2);
        anbMap.set('b', 1);
      }
      if (nthName.indexOf('last') > -1) {
        anbMap.set('reverse', true);
      }
    } else {
      if (typeof a === 'string' && /-?\d+/.test(a)) {
        anbMap.set('a', a * 1);
      } else {
        anbMap.set('a', 0);
      }
      if (typeof b === 'string' && /-?\d+/.test(b)) {
        anbMap.set('b', b * 1);
      } else {
        anbMap.set('b', 0);
      }
      if (nthName.indexOf('last') > -1) {
        anbMap.set('reverse', true);
      }
    }
    let matched = new Set();
    if (anbMap.has('a') && anbMap.has('b')) {
      if (/^nth-(?:last-)?child$/.test(nthName)) {
        if (selector) {
          anbMap.set('selector', selector);
        }
        const anb = Object.fromEntries(anbMap);
        const nodes = this._collectNthChild(anb, node);
        if (nodes.size) {
          matched = nodes;
        }
      } else if (/^nth-(?:last-)?of-type$/.test(nthName)) {
        const anb = Object.fromEntries(anbMap);
        const nodes = this._collectNthOfType(anb, node);
        if (nodes.size) {
          matched = nodes;
        }
      }
    }
    return matched;
  }

  /**
   * match pseudo element selector
   * @param {string} astName - AST name
   * @param {object} [opt] - options
   * @param {boolean} [opt.forgive] - is forgiving selector list
   * @throws {DOMException}
   * @returns {void}
   */
  _matchPseudoElementSelector(astName, opt = {}) {
    const { forgive } = opt;
    switch (astName) {
      case 'after':
      case 'backdrop':
      case 'before':
      case 'cue':
      case 'cue-region':
      case 'first-letter':
      case 'first-line':
      case 'file-selector-button':
      case 'marker':
      case 'placeholder':
      case 'selection':
      case 'target-text': {
        if (this.#warn) {
          const msg = `Unsupported pseudo-element ::${astName}`;
          throw new DOMException(msg, NOT_SUPPORTED_ERR);
        }
        break;
      }
      case 'part':
      case 'slotted': {
        if (this.#warn) {
          const msg = `Unsupported pseudo-element ::${astName}()`;
          throw new DOMException(msg, NOT_SUPPORTED_ERR);
        }
        break;
      }
      default: {
        if (astName.startsWith('-webkit-')) {
          if (this.#warn) {
            const msg = `Unsupported pseudo-element ::${astName}`;
            throw new DOMException(msg, NOT_SUPPORTED_ERR);
          }
        } else if (!forgive) {
          const msg = `Unknown pseudo-element ::${astName}`;
          throw new DOMException(msg, SYNTAX_ERR);
        }
      }
    }
  }

  /**
   * match directionality pseudo-class - :dir()
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @returns {?object} - matched node
   */
  _matchDirectionPseudoClass(ast, node) {
    const astName = unescapeSelector(ast.name);
    const dir = getDirectionality(node);
    let res;
    if (astName === dir) {
      res = node;
    }
    return res ?? null;
  }

  /**
   * match language pseudo-class - :lang()
   * @see https://datatracker.ietf.org/doc/html/rfc4647#section-3.3.1
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @returns {?object} - matched node
   */
  _matchLanguagePseudoClass(ast, node) {
    const astName = unescapeSelector(ast.name);
    let res;
    if (astName === '*') {
      if (node.hasAttribute('lang')) {
        if (node.getAttribute('lang')) {
          res = node;
        }
      } else {
        let parent = node.parentNode;
        while (parent) {
          if (parent.nodeType === ELEMENT_NODE) {
            if (parent.hasAttribute('lang')) {
              if (parent.getAttribute('lang')) {
                res = node;
              }
              break;
            }
            parent = parent.parentNode;
          } else {
            break;
          }
        }
      }
    } else if (astName) {
      const langPart = `(?:-${ALPHA_NUM})*`;
      const regLang = new RegExp(`^(?:\\*-)?${ALPHA_NUM}${langPart}$`, 'i');
      if (regLang.test(astName)) {
        let regExtendedLang;
        if (astName.indexOf('-') > -1) {
          const [langMain, langSub, ...langRest] = astName.split('-');
          let extendedMain;
          if (langMain === '*') {
            extendedMain = `${ALPHA_NUM}${langPart}`;
          } else {
            extendedMain = `${langMain}${langPart}`;
          }
          const extendedSub = `-${langSub}${langPart}`;
          const len = langRest.length;
          let extendedRest = '';
          if (len) {
            for (let i = 0; i < len; i++) {
              extendedRest += `-${langRest[i]}${langPart}`;
            }
          }
          regExtendedLang =
            new RegExp(`^${extendedMain}${extendedSub}${extendedRest}$`, 'i');
        } else {
          regExtendedLang = new RegExp(`^${astName}${langPart}$`, 'i');
        }
        if (node.hasAttribute('lang')) {
          if (regExtendedLang.test(node.getAttribute('lang'))) {
            res = node;
          }
        } else {
          let parent = node.parentNode;
          while (parent) {
            if (parent.nodeType === ELEMENT_NODE) {
              if (parent.hasAttribute('lang')) {
                const value = parent.getAttribute('lang');
                if (regExtendedLang.test(value)) {
                  res = node;
                }
                break;
              }
              parent = parent.parentNode;
            } else {
              break;
            }
          }
        }
      }
    }
    return res ?? null;
  }

  /**
   * match :has() pseudo-class function
   * @param {Array.<object>} leaves - AST leaves
   * @param {object} node - Element node
   * @returns {boolean} - result
   */
  _matchHasPseudoFunc(leaves, node) {
    let bool;
    if (Array.isArray(leaves) && leaves.length) {
      const [leaf] = leaves;
      const { type: leafType } = leaf;
      let combo;
      if (leafType === COMBINATOR) {
        combo = leaves.shift();
      } else {
        combo = {
          name: ' ',
          type: COMBINATOR
        };
      }
      const twigLeaves = [];
      while (leaves.length) {
        const [item] = leaves;
        const { type: itemType } = item;
        if (itemType === COMBINATOR) {
          break;
        } else {
          twigLeaves.push(leaves.shift());
        }
      }
      const twig = {
        combo,
        leaves: twigLeaves
      };
      const nodes = this._matchCombinator(twig, node, {
        dir: DIR_NEXT
      });
      if (nodes.size) {
        if (leaves.length) {
          for (const nextNode of nodes) {
            bool =
              this._matchHasPseudoFunc(Object.assign([], leaves), nextNode);
            if (bool) {
              break;
            }
          }
        } else {
          bool = true;
        }
      }
    }
    return !!bool;
  }

  /**
   * match logical pseudo-class functions - :has(), :is(), :not(), :where()
   * @param {object} astData - AST data
   * @param {object} node - Element node
   * @returns {?object} - matched node
   */
  _matchLogicalPseudoFunc(astData, node) {
    const {
      astName = '', branches = [], selector = '', twigBranches = []
    } = astData;
    let res;
    if (astName === 'has') {
      if (selector.includes(':has(')) {
        res = null;
      } else {
        let bool;
        for (const leaves of branches) {
          bool = this._matchHasPseudoFunc(Object.assign([], leaves), node);
          if (bool) {
            break;
          }
        }
        if (bool) {
          res = node;
        }
      }
    } else {
      const forgive = /^(?:is|where)$/.test(astName);
      const l = twigBranches.length;
      let bool;
      for (let i = 0; i < l; i++) {
        const branch = twigBranches[i];
        const lastIndex = branch.length - 1;
        const { leaves } = branch[lastIndex];
        bool = this._matchLeaves(leaves, node, { forgive });
        if (bool && lastIndex > 0) {
          let nextNodes = new Set([node]);
          for (let j = lastIndex - 1; j >= 0; j--) {
            const twig = branch[j];
            const arr = [];
            for (const nextNode of nextNodes) {
              const m = this._matchCombinator(twig, nextNode, {
                forgive,
                dir: DIR_PREV
              });
              if (m.size) {
                arr.push(...m);
              }
            }
            if (arr.length) {
              if (j === 0) {
                bool = true;
              } else {
                nextNodes = new Set(arr);
              }
            } else {
              bool = false;
              break;
            }
          }
        }
        if (bool) {
          break;
        }
      }
      if (astName === 'not') {
        if (!bool) {
          res = node;
        }
      } else if (bool) {
        res = node;
      }
    }
    return res ?? null;
  }

  /**
   * match pseudo-class selector
   * @see https://html.spec.whatwg.org/#pseudo-classes
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @param {object} [opt] - options
   * @param {boolean} [opt.forgive] - is forgiving selector list
   * @returns {Set.<object>} - collection of matched nodes
   */
  _matchPseudoClassSelector(ast, node, opt = {}) {
    const { children: astChildren } = ast;
    const { localName, parentNode } = node;
    const { forgive } = opt;
    const astName = unescapeSelector(ast.name);
    let matched = new Set();
    // :has(), :is(), :not(), :where()
    if (REG_LOGICAL_PSEUDO.test(astName)) {
      let astData;
      if (this.#cache.has(ast)) {
        astData = this.#cache.get(ast);
      } else {
        const branches = walkAST(ast);
        const selectors = [];
        const twigBranches = [];
        for (const [...leaves] of branches) {
          for (const leaf of leaves) {
            const css = generateCSS(leaf);
            selectors.push(css);
          }
          const branch = [];
          const leavesSet = new Set();
          let item = leaves.shift();
          while (item) {
            if (item.type === COMBINATOR) {
              branch.push({
                combo: item,
                leaves: [...leavesSet]
              });
              leavesSet.clear();
            } else if (item) {
              leavesSet.add(item);
            }
            if (leaves.length) {
              item = leaves.shift();
            } else {
              branch.push({
                combo: null,
                leaves: [...leavesSet]
              });
              leavesSet.clear();
              break;
            }
          }
          twigBranches.push(branch);
        }
        astData = {
          astName,
          branches,
          twigBranches,
          selector: selectors.join(',')
        };
        this.#cache.set(ast, astData);
      }
      const res = this._matchLogicalPseudoFunc(astData, node);
      if (res) {
        matched.add(res);
      }
    } else if (Array.isArray(astChildren)) {
      const [branch] = astChildren;
      // :nth-child(), :nth-last-child(), nth-of-type(), :nth-last-of-type()
      if (/^nth-(?:last-)?(?:child|of-type)$/.test(astName)) {
        const nodes = this._matchAnPlusB(branch, node, astName);
        if (nodes.size) {
          matched = nodes;
        }
      // :dir()
      } else if (astName === 'dir') {
        const res = this._matchDirectionPseudoClass(branch, node);
        if (res) {
          matched.add(res);
        }
      // :lang()
      } else if (astName === 'lang') {
        const res = this._matchLanguagePseudoClass(branch, node);
        if (res) {
          matched.add(res);
        }
      } else {
        switch (astName) {
          case 'current':
          case 'nth-col':
          case 'nth-last-col': {
            if (this.#warn) {
              const msg = `Unsupported pseudo-class :${astName}()`;
              throw new DOMException(msg, NOT_SUPPORTED_ERR);
            }
            break;
          }
          case 'host':
          case 'host-context': {
            // ignore
            break;
          }
          default: {
            if (!forgive) {
              const msg = `Unknown pseudo-class :${astName}()`;
              throw new DOMException(msg, SYNTAX_ERR);
            }
          }
        }
      }
    } else {
      const regAnchor = /^a(?:rea)?$/;
      const regFormCtrl =
        /^(?:(?:fieldse|inpu|selec)t|button|opt(?:group|ion)|textarea)$/;
      const regFormValidity = /^(?:(?:inpu|selec)t|button|form|textarea)$/;
      const regInteract = /^d(?:etails|ialog)$/;
      const regTypeCheck = /^(?:checkbox|radio)$/;
      const regTypeDate = /^(?:date(?:time-local)?|month|time|week)$/;
      const regTypeRange =
        /(?:(?:rang|tim)e|date(?:time-local)?|month|number|week)$/;
      const regTypeText = /^(?:(?:emai|te|ur)l|number|password|search|text)$/;
      switch (astName) {
        case 'any-link':
        case 'link': {
          if (regAnchor.test(localName) && node.hasAttribute('href')) {
            matched.add(node);
          }
          break;
        }
        case 'local-link': {
          if (regAnchor.test(localName) && node.hasAttribute('href')) {
            const { href, origin, pathname } = new URL(this.#document.URL);
            const attrURL = new URL(node.getAttribute('href'), href);
            if (attrURL.origin === origin && attrURL.pathname === pathname) {
              matched.add(node);
            }
          }
          break;
        }
        case 'visited': {
          // prevent fingerprinting
          break;
        }
        case 'target': {
          const { hash } = new URL(this.#document.URL);
          if (node.id && hash === `#${node.id}` &&
              this.#document.contains(node)) {
            matched.add(node);
          }
          break;
        }
        case 'target-within': {
          const { hash } = new URL(this.#document.URL);
          if (hash) {
            const id = hash.replace(/^#/, '');
            let current = this.#document.getElementById(id);
            while (current) {
              if (current === node) {
                matched.add(node);
                break;
              }
              current = current.parentNode;
            }
          }
          break;
        }
        case 'scope': {
          if (this.#node.nodeType === ELEMENT_NODE) {
            if (node === this.#node) {
              matched.add(node);
            }
          } else if (node === this.#document.documentElement) {
            matched.add(node);
          }
          break;
        }
        case 'focus': {
          if (node === this.#document.activeElement) {
            matched.add(node);
          }
          break;
        }
        case 'focus-within': {
          let current = this.#document.activeElement;
          while (current) {
            if (current === node) {
              matched.add(node);
              break;
            }
            current = current.parentNode;
          }
          break;
        }
        case 'open': {
          if (regInteract.test(localName) && node.hasAttribute('open')) {
            matched.add(node);
          }
          break;
        }
        case 'closed': {
          if (regInteract.test(localName) && !node.hasAttribute('open')) {
            matched.add(node);
          }
          break;
        }
        case 'disabled': {
          if (regFormCtrl.test(localName) || isCustomElementName(localName)) {
            if (node.disabled || node.hasAttribute('disabled')) {
              matched.add(node);
            } else {
              let parent = parentNode;
              while (parent) {
                if (parent.localName === 'fieldset') {
                  break;
                }
                parent = parent.parentNode;
              }
              if (parent && parentNode.localName !== 'legend' &&
                  parent.hasAttribute('disabled')) {
                matched.add(node);
              }
            }
          }
          break;
        }
        case 'enabled': {
          if ((regFormCtrl.test(localName) || isCustomElementName(localName)) &&
              !(node.disabled && node.hasAttribute('disabled'))) {
            matched.add(node);
          }
          break;
        }
        case 'read-only': {
          switch (localName) {
            case 'textarea': {
              if (node.readonly || node.hasAttribute('readonly') ||
                  node.disabled || node.hasAttribute('disabled')) {
                matched.add(node);
              }
              break;
            }
            case 'input': {
              if ((!node.type || regTypeDate.test(node.type) ||
                   regTypeText.test(node.type)) &&
                  (node.readonly || node.hasAttribute('readonly') ||
                   node.disabled || node.hasAttribute('disabled'))) {
                matched.add(node);
              }
              break;
            }
            default: {
              if (!isContentEditable(node)) {
                matched.add(node);
              }
            }
          }
          break;
        }
        case 'read-write': {
          switch (localName) {
            case 'textarea': {
              if (!(node.readonly || node.hasAttribute('readonly') ||
                    node.disabled || node.hasAttribute('disabled'))) {
                matched.add(node);
              }
              break;
            }
            case 'input': {
              if ((!node.type || regTypeDate.test(node.type) ||
                   regTypeText.test(node.type)) &&
                  !(node.readonly || node.hasAttribute('readonly') ||
                    node.disabled || node.hasAttribute('disabled'))) {
                matched.add(node);
              }
              break;
            }
            default: {
              if (isContentEditable(node)) {
                matched.add(node);
              }
            }
          }
          break;
        }
        case 'placeholder-shown': {
          let targetNode;
          if (localName === 'textarea') {
            targetNode = node;
          } else if (localName === 'input') {
            if (node.hasAttribute('type')) {
              if (regTypeText.test(node.getAttribute('type'))) {
                targetNode = node;
              }
            } else {
              targetNode = node;
            }
          }
          if (targetNode && node.value === '' &&
              node.hasAttribute('placeholder') &&
              node.getAttribute('placeholder').trim().length) {
            matched.add(node);
          }
          break;
        }
        case 'checked': {
          if ((node.checked && localName === 'input' &&
               node.hasAttribute('type') &&
               regTypeCheck.test(node.getAttribute('type'))) ||
              (node.selected && localName === 'option')) {
            matched.add(node);
          }
          break;
        }
        case 'indeterminate': {
          if ((node.indeterminate && localName === 'input' &&
               node.type === 'checkbox') ||
              (localName === 'progress' && !node.hasAttribute('value'))) {
            matched.add(node);
          } else if (localName === 'input' && node.type === 'radio' &&
                     !node.hasAttribute('checked')) {
            const nodeName = node.name;
            let parent = node.parentNode;
            while (parent) {
              if (parent.localName === 'form') {
                break;
              }
              parent = parent.parentNode;
            }
            if (!parent) {
              parent = this.#document.documentElement;
            }
            let checked;
            const nodes = [].slice.call(parent.getElementsByTagName('input'));
            for (const item of nodes) {
              if (item.getAttribute('type') === 'radio') {
                if (nodeName) {
                  if (item.getAttribute('name') === nodeName) {
                    checked = !!item.checked;
                  }
                } else if (!item.hasAttribute('name')) {
                  checked = !!item.checked;
                }
                if (checked) {
                  break;
                }
              }
            }
            if (!checked) {
              matched.add(node);
            }
          }
          break;
        }
        case 'default': {
          const regTypeReset = /^(?:button|reset)$/;
          const regTypeSubmit = /^(?:image|submit)$/;
          // button[type="submit"], input[type="submit"], input[type="image"]
          if ((localName === 'button' &&
               !(node.hasAttribute('type') &&
                 regTypeReset.test(node.getAttribute('type')))) ||
              (localName === 'input' && node.hasAttribute('type') &&
               regTypeSubmit.test(node.getAttribute('type')))) {
            let form = node.parentNode;
            while (form) {
              if (form.localName === 'form') {
                break;
              }
              form = form.parentNode;
            }
            if (form) {
              const walker =
                this.#document.createTreeWalker(form, SHOW_ELEMENT);
              let nextNode = walker.firstChild();
              while (nextNode) {
                const nodeName = nextNode.localName;
                let m;
                if (nodeName === 'button') {
                  m = !(nextNode.hasAttribute('type') &&
                    regTypeReset.test(nextNode.getAttribute('type')));
                } else if (nodeName === 'input') {
                  m = nextNode.hasAttribute('type') &&
                    regTypeSubmit.test(nextNode.getAttribute('type'));
                }
                if (m) {
                  if (nextNode === node) {
                    matched.add(node);
                  }
                  break;
                }
                nextNode = walker.nextNode();
              }
            }
          // input[type="checkbox"], input[type="radio"]
          } else if (localName === 'input' && node.hasAttribute('type') &&
                     regTypeCheck.test(node.getAttribute('type')) &&
                     (node.checked || node.hasAttribute('checked'))) {
            matched.add(node);
          // option
          } else if (localName === 'option') {
            let isMultiple = false;
            let parent = parentNode;
            while (parent) {
              if (parent.localName === 'datalist') {
                break;
              } else if (parent.localName === 'select') {
                if (parent.multiple || parent.hasAttribute('multiple')) {
                  isMultiple = true;
                }
                break;
              }
              parent = parent.parentNode;
            }
            if (isMultiple) {
              if (node.selected || node.hasAttribute('selected')) {
                matched.add(node);
              }
            } else {
              const defaultOpt = new Set();
              const walker =
                this.#document.createTreeWalker(parentNode, SHOW_ELEMENT);
              let refNode = walker.firstChild();
              while (refNode) {
                if (refNode.selected || refNode.hasAttribute('selected')) {
                  defaultOpt.add(refNode);
                  break;
                }
                refNode = walker.nextSibling();
              }
              if (defaultOpt.size) {
                if (defaultOpt.has(node)) {
                  matched.add(node);
                }
              }
            }
          }
          break;
        }
        case 'valid': {
          if (regFormValidity.test(localName)) {
            if (node.checkValidity()) {
              matched.add(node);
            }
          } else if (localName === 'fieldset') {
            let bool;
            const walker = this.#document.createTreeWalker(node, SHOW_ELEMENT);
            let refNode = walker.firstChild();
            while (refNode) {
              if (regFormValidity.test(refNode.localName)) {
                bool = refNode.checkValidity();
                if (!bool) {
                  break;
                }
              }
              refNode = walker.nextNode();
            }
            if (bool) {
              matched.add(node);
            }
          }
          break;
        }
        case 'invalid': {
          if (regFormValidity.test(localName)) {
            if (!node.checkValidity()) {
              matched.add(node);
            }
          } else if (localName === 'fieldset') {
            let bool;
            const walker = this.#document.createTreeWalker(node, SHOW_ELEMENT);
            let refNode = walker.firstChild();
            while (refNode) {
              if (regFormValidity.test(refNode.localName)) {
                bool = refNode.checkValidity();
                if (!bool) {
                  break;
                }
              }
              refNode = walker.nextNode();
            }
            if (!bool) {
              matched.add(node);
            }
          }
          break;
        }
        case 'in-range': {
          if (localName === 'input' &&
              !(node.readonly || node.hasAttribute('readonly')) &&
              !(node.disabled || node.hasAttribute('disabled')) &&
              node.hasAttribute('type') &&
              regTypeRange.test(node.getAttribute('type')) &&
              !(node.validity.rangeUnderflow ||
                node.validity.rangeOverflow) &&
              (node.hasAttribute('min') || node.hasAttribute('max') ||
               node.getAttribute('type') === 'range')) {
            matched.add(node);
          }
          break;
        }
        case 'out-of-range': {
          if (localName === 'input' &&
              !(node.readonly || node.hasAttribute('readonly')) &&
              !(node.disabled || node.hasAttribute('disabled')) &&
              node.hasAttribute('type') &&
              regTypeRange.test(node.getAttribute('type')) &&
              (node.validity.rangeUnderflow || node.validity.rangeOverflow)) {
            matched.add(node);
          }
          break;
        }
        case 'required': {
          let targetNode;
          if (/^(?:select|textarea)$/.test(localName)) {
            targetNode = node;
          } else if (localName === 'input') {
            if (node.hasAttribute('type')) {
              const inputType = node.getAttribute('type');
              if (inputType === 'file' || regTypeCheck.test(inputType) ||
                  regTypeDate.test(inputType) || regTypeText.test(inputType)) {
                targetNode = node;
              }
            } else {
              targetNode = node;
            }
          }
          if (targetNode &&
              (node.required || node.hasAttribute('required'))) {
            matched.add(node);
          }
          break;
        }
        case 'optional': {
          let targetNode;
          if (/^(?:select|textarea)$/.test(localName)) {
            targetNode = node;
          } else if (localName === 'input') {
            if (node.hasAttribute('type')) {
              const inputType = node.getAttribute('type');
              if (inputType === 'file' || regTypeCheck.test(inputType) ||
                  regTypeDate.test(inputType) || regTypeText.test(inputType)) {
                targetNode = node;
              }
            } else {
              targetNode = node;
            }
          }
          if (targetNode &&
              !(node.required || node.hasAttribute('required'))) {
            matched.add(node);
          }
          break;
        }
        case 'root': {
          if (node === this.#document.documentElement) {
            matched.add(node);
          }
          break;
        }
        case 'empty': {
          if (node.hasChildNodes()) {
            let bool;
            const walker = this.#document.createTreeWalker(node, SHOW_ALL);
            let refNode = walker.firstChild();
            while (refNode) {
              bool = refNode.nodeType !== ELEMENT_NODE &&
                refNode.nodeType !== TEXT_NODE;
              if (!bool) {
                break;
              }
              refNode = walker.nextSibling();
            }
            if (bool) {
              matched.add(node);
            }
          } else {
            matched.add(node);
          }
          break;
        }
        case 'first-child': {
          if ((parentNode && node === parentNode.firstElementChild) ||
              (node === this.#root && this.#root.nodeType === ELEMENT_NODE)) {
            matched.add(node);
          }
          break;
        }
        case 'last-child': {
          if ((parentNode && node === parentNode.lastElementChild) ||
              (node === this.#root && this.#root.nodeType === ELEMENT_NODE)) {
            matched.add(node);
          }
          break;
        }
        case 'only-child': {
          if ((parentNode &&
               node === parentNode.firstElementChild &&
               node === parentNode.lastElementChild) ||
              (node === this.#root && this.#root.nodeType === ELEMENT_NODE)) {
            matched.add(node);
          }
          break;
        }
        case 'first-of-type': {
          if (parentNode) {
            const [node1] = this._collectNthOfType({
              a: 0,
              b: 1
            }, node);
            if (node1) {
              matched.add(node1);
            }
          } else if (node === this.#root &&
                     this.#root.nodeType === ELEMENT_NODE) {
            matched.add(node);
          }
          break;
        }
        case 'last-of-type': {
          if (parentNode) {
            const [node1] = this._collectNthOfType({
              a: 0,
              b: 1,
              reverse: true
            }, node);
            if (node1) {
              matched.add(node1);
            }
          } else if (node === this.#root &&
                     this.#root.nodeType === ELEMENT_NODE) {
            matched.add(node);
          }
          break;
        }
        case 'only-of-type': {
          if (parentNode) {
            const [node1] = this._collectNthOfType({
              a: 0,
              b: 1
            }, node);
            if (node1 === node) {
              const [node2] = this._collectNthOfType({
                a: 0,
                b: 1,
                reverse: true
              }, node);
              if (node2 === node) {
                matched.add(node);
              }
            }
          } else if (node === this.#root &&
                     this.#root.nodeType === ELEMENT_NODE) {
            matched.add(node);
          }
          break;
        }
        case 'host':
        case 'host-context': {
          // ignore
          break;
        }
        // legacy pseudo-elements
        case 'after':
        case 'before':
        case 'first-letter':
        case 'first-line': {
          if (this.#warn) {
            const msg = `Unsupported pseudo-element ::${astName}`;
            throw new DOMException(msg, NOT_SUPPORTED_ERR);
          }
          break;
        }
        case 'active':
        case 'autofill':
        case 'blank':
        case 'buffering':
        case 'current':
        case 'defined':
        case 'focus-visible':
        case 'fullscreen':
        case 'future':
        case 'hover':
        case 'modal':
        case 'muted':
        case 'past':
        case 'paused':
        case 'picture-in-picture':
        case 'playing':
        case 'seeking':
        case 'stalled':
        case 'user-invalid':
        case 'user-valid':
        case 'volume-locked':
        case '-webkit-autofill': {
          if (this.#warn) {
            const msg = `Unsupported pseudo-class :${astName}`;
            throw new DOMException(msg, NOT_SUPPORTED_ERR);
          }
          break;
        }
        default: {
          if (astName.startsWith('-webkit-')) {
            if (this.#warn) {
              const msg = `Unsupported pseudo-class :${astName}`;
              throw new DOMException(msg, NOT_SUPPORTED_ERR);
            }
          } else if (!forgive) {
            const msg = `Unknown pseudo-class :${astName}`;
            throw new DOMException(msg, SYNTAX_ERR);
          }
        }
      }
    }
    return matched;
  }

  /**
   * match attribute selector
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @returns {?object} - matched node
   */
  _matchAttributeSelector(ast, node) {
    const {
      flags: astFlags, matcher: astMatcher, name: astName, value: astValue
    } = ast;
    if (typeof astFlags === 'string' && !/^[is]$/i.test(astFlags)) {
      const css = generateCSS(ast);
      const msg = `Invalid selector ${css}`;
      throw new DOMException(msg, SYNTAX_ERR);
    }
    const { attributes } = node;
    let res;
    if (attributes && attributes.length) {
      let caseInsensitive;
      if (this.#document.contentType === 'text/html') {
        if (typeof astFlags === 'string' && /^s$/i.test(astFlags)) {
          caseInsensitive = false;
        } else {
          caseInsensitive = true;
        }
      } else if (typeof astFlags === 'string' && /^i$/i.test(astFlags)) {
        caseInsensitive = true;
      } else {
        caseInsensitive = false;
      }
      let astAttrName = unescapeSelector(astName.name);
      if (caseInsensitive) {
        astAttrName = astAttrName.toLowerCase();
      }
      const attrValues = new Set();
      // namespaced
      if (astAttrName.indexOf('|') > -1) {
        const {
          prefix: astAttrPrefix, tagName: astAttrLocalName
        } = selectorToNodeProps(astAttrName);
        for (let { name: itemName, value: itemValue } of attributes) {
          if (caseInsensitive) {
            itemName = itemName.toLowerCase();
            itemValue = itemValue.toLowerCase();
          }
          switch (astAttrPrefix) {
            case '': {
              if (astAttrLocalName === itemName) {
                attrValues.add(itemValue);
              }
              break;
            }
            case '*': {
              if (itemName.indexOf(':') > -1) {
                if (itemName.endsWith(`:${astAttrLocalName}`)) {
                  attrValues.add(itemValue);
                }
              } else if (astAttrLocalName === itemName) {
                attrValues.add(itemValue);
              }
              break;
            }
            default: {
              if (itemName.indexOf(':') > -1) {
                const [itemNamePrefix, itemNameLocalName] = itemName.split(':');
                if (astAttrPrefix === itemNamePrefix &&
                    astAttrLocalName === itemNameLocalName &&
                    isNamespaceDeclared(astAttrPrefix, node)) {
                  attrValues.add(itemValue);
                }
              }
            }
          }
        }
      } else {
        for (let { name: itemName, value: itemValue } of attributes) {
          if (caseInsensitive) {
            itemName = itemName.toLowerCase();
            itemValue = itemValue.toLowerCase();
          }
          if (itemName.indexOf(':') > -1) {
            const [itemNamePrefix, itemNameLocalName] = itemName.split(':');
            // ignore xml:lang
            if (itemNamePrefix === 'xml' && itemNameLocalName === 'lang') {
              continue;
            } else if (astAttrName === itemNameLocalName) {
              attrValues.add(itemValue);
            }
          } else if (astAttrName === itemName) {
            attrValues.add(itemValue);
          }
        }
      }
      if (attrValues.size) {
        const {
          name: astAttrIdentValue, value: astAttrStringValue
        } = astValue || {};
        let attrValue;
        if (astAttrIdentValue) {
          if (caseInsensitive) {
            attrValue = astAttrIdentValue.toLowerCase();
          } else {
            attrValue = astAttrIdentValue;
          }
        } else if (astAttrStringValue) {
          if (caseInsensitive) {
            attrValue = astAttrStringValue.toLowerCase();
          } else {
            attrValue = astAttrStringValue;
          }
        } else if (astAttrStringValue === '') {
          attrValue = astAttrStringValue;
        }
        switch (astMatcher) {
          case '=': {
            if (typeof attrValue === 'string' && attrValues.has(attrValue)) {
              res = node;
            }
            break;
          }
          case '~=': {
            if (attrValue && typeof attrValue === 'string') {
              for (const value of attrValues) {
                const item = new Set(value.split(/\s+/));
                if (item.has(attrValue)) {
                  res = node;
                  break;
                }
              }
            }
            break;
          }
          case '|=': {
            if (attrValue && typeof attrValue === 'string') {
              let item;
              for (const value of attrValues) {
                if (value === attrValue || value.startsWith(`${attrValue}-`)) {
                  item = value;
                  break;
                }
              }
              if (item) {
                res = node;
              }
            }
            break;
          }
          case '^=': {
            if (attrValue && typeof attrValue === 'string') {
              let item;
              for (const value of attrValues) {
                if (value.startsWith(`${attrValue}`)) {
                  item = value;
                  break;
                }
              }
              if (item) {
                res = node;
              }
            }
            break;
          }
          case '$=': {
            if (attrValue && typeof attrValue === 'string') {
              let item;
              for (const value of attrValues) {
                if (value.endsWith(`${attrValue}`)) {
                  item = value;
                  break;
                }
              }
              if (item) {
                res = node;
              }
            }
            break;
          }
          case '*=': {
            if (attrValue && typeof attrValue === 'string') {
              let item;
              for (const value of attrValues) {
                if (value.includes(`${attrValue}`)) {
                  item = value;
                  break;
                }
              }
              if (item) {
                res = node;
              }
            }
            break;
          }
          case null:
          default: {
            res = node;
          }
        }
      }
    }
    return res ?? null;
  }

  /**
   * match class selector
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @returns {?object} - matched node
   */
  _matchClassSelector(ast, node) {
    const astName = unescapeSelector(ast.name);
    let res;
    if (node.classList.contains(astName)) {
      res = node;
    }
    return res ?? null;
  }

  /**
   * match ID selector
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @returns {?object} - matched node
   */
  _matchIDSelector(ast, node) {
    const astName = unescapeSelector(ast.name);
    const { id } = node;
    let res;
    if (astName === id) {
      res = node;
    }
    return res ?? null;
  }

  /**
   * match type selector
   * @param {object} ast - AST
   * @param {object} node - Element node
   * @param {object} [opt] - options
   * @param {boolean} [opt.forgive] - is forgiving selector list
   * @returns {?object} - matched node
   */
  _matchTypeSelector(ast, node, opt = {}) {
    const astName = unescapeSelector(ast.name);
    const { localName, prefix } = node;
    const { forgive } = opt;
    let {
      prefix: astPrefix, tagName: astNodeName
    } = selectorToNodeProps(astName, node);
    if (this.#document.contentType === 'text/html') {
      astPrefix = astPrefix.toLowerCase();
      astNodeName = astNodeName.toLowerCase();
    }
    let nodePrefix;
    let nodeName;
    // just in case that the namespaced content is parsed as text/html
    if (localName.indexOf(':') > -1) {
      [nodePrefix, nodeName] = localName.split(':');
    } else {
      nodePrefix = prefix || '';
      nodeName = localName;
    }
    let res;
    if (astPrefix === '' && nodePrefix === '') {
      if (node.namespaceURI === null &&
          (astNodeName === '*' || astNodeName === nodeName)) {
        res = node;
      }
    } else if (astPrefix === '*') {
      if (astNodeName === '*' || astNodeName === nodeName) {
        res = node;
      }
    } else if (astPrefix === nodePrefix) {
      if (isNamespaceDeclared(astPrefix, node)) {
        if (astNodeName === '*' || astNodeName === nodeName) {
          res = node;
        }
      } else if (!forgive) {
        const msg = `Undeclared namespace ${astPrefix}`;
        throw new DOMException(msg, SYNTAX_ERR);
      }
    } else if (astPrefix && !forgive && !isNamespaceDeclared(astPrefix, node)) {
      const msg = `Undeclared namespace ${astPrefix}`;
      throw new DOMException(msg, SYNTAX_ERR);
    }
    return res ?? null;
  };

  /**
   * match shadow host pseudo class
   * @param {object} ast - AST
   * @param {object} node - DocumentFragment node
   * @returns {?object} - matched node
   */
  _matchShadowHostPseudoClass(ast, node) {
    const { children: astChildren } = ast;
    const astName = unescapeSelector(ast.name);
    let res;
    if (Array.isArray(astChildren)) {
      const [branch] = walkAST(astChildren[0]);
      const [...leaves] = branch;
      const { host } = node;
      if (astName === 'host') {
        let bool;
        for (const leaf of leaves) {
          const { type: leafType } = leaf;
          if (leafType === COMBINATOR) {
            const css = generateCSS(ast);
            const msg = `Invalid selector ${css}`;
            throw new DOMException(msg, SYNTAX_ERR);
          }
          bool = this._matchSelector(leaf, host).has(host);
          if (!bool) {
            break;
          }
        }
        if (bool) {
          res = node;
        }
      } else if (astName === 'host-context') {
        let bool;
        let parent = host;
        while (parent) {
          for (const leaf of leaves) {
            const { type: leafType } = leaf;
            if (leafType === COMBINATOR) {
              const css = generateCSS(ast);
              const msg = `Invalid selector ${css}`;
              throw new DOMException(msg, SYNTAX_ERR);
            }
            bool = this._matchSelector(leaf, parent).has(parent);
            if (!bool) {
              break;
            }
          }
          if (bool) {
            break;
          } else {
            parent = parent.parentNode;
          }
        }
        if (bool) {
          res = node;
        }
      }
    } else if (astName === 'host') {
      res = node;
    } else {
      const msg = `Invalid selector :${astName}`;
      throw new DOMException(msg, SYNTAX_ERR);
    }
    return res ?? null;
  }

  /**
   * match selector
   * @param {object} ast - AST
   * @param {object} node - Document, DocumentFragment, Element node
   * @param {object} [opt] - options
   * @returns {Set.<object>} - collection of matched nodes
   */
  _matchSelector(ast, node, opt) {
    const { type: astType } = ast;
    const astName = unescapeSelector(ast.name);
    let matched = new Set();
    if (node.nodeType === ELEMENT_NODE) {
      switch (astType) {
        case SELECTOR_ATTR: {
          const res = this._matchAttributeSelector(ast, node);
          if (res) {
            matched.add(res);
          }
          break;
        }
        case SELECTOR_CLASS: {
          const res = this._matchClassSelector(ast, node);
          if (res) {
            matched.add(res);
          }
          break;
        }
        case SELECTOR_ID: {
          const res = this._matchIDSelector(ast, node);
          if (res) {
            matched.add(res);
          }
          break;
        }
        case SELECTOR_PSEUDO_CLASS: {
          const nodes = this._matchPseudoClassSelector(ast, node, opt);
          if (nodes.size) {
            matched = nodes;
          }
          break;
        }
        case SELECTOR_PSEUDO_ELEMENT: {
          this._matchPseudoElementSelector(astName, opt);
          break;
        }
        case SELECTOR_TYPE:
        default: {
          const res = this._matchTypeSelector(ast, node, opt);
          if (res) {
            matched.add(res);
          }
        }
      }
    } else if (this.#shadow && astType === SELECTOR_PSEUDO_CLASS &&
               node.nodeType === DOCUMENT_FRAGMENT_NODE) {
      if (astName !== 'has' && REG_LOGICAL_PSEUDO.test(astName)) {
        const nodes = this._matchPseudoClassSelector(ast, node, opt);
        if (nodes.size) {
          matched = nodes;
        }
      } else if (REG_SHADOW_HOST.test(astName)) {
        const res = this._matchShadowHostPseudoClass(ast, node);
        if (res) {
          matched.add(res);
        }
      }
    }
    return matched;
  }

  /**
   * match leaves
   * @param {Array.<object>} leaves - AST leaves
   * @param {object} node - node
   * @param {object} [opt] - options
   * @returns {boolean} - result
   */
  _matchLeaves(leaves, node, opt) {
    let bool;
    for (const leaf of leaves) {
      bool = this._matchSelector(leaf, node, opt).has(node);
      if (!bool) {
        break;
      }
    }
    return !!bool;
  }

  /**
   * find descendant nodes
   * @param {Array.<object>} leaves - AST leaves
   * @param {object} baseNode - base Element node
   * @returns {object} - collection of nodes and pending state
   */
  _findDescendantNodes(leaves, baseNode) {
    const [leaf, ...filterLeaves] = leaves;
    const { type: leafType } = leaf;
    const leafName = unescapeSelector(leaf.name);
    const compound = filterLeaves.length > 0;
    let nodes = new Set();
    let pending = false;
    if (this.#shadow) {
      pending = true;
    } else {
      switch (leafType) {
        case SELECTOR_ID: {
          if (this.#root.nodeType === ELEMENT_NODE) {
            pending = true;
          } else {
            const node = this.#root.getElementById(leafName);
            if (node && node !== baseNode && baseNode.contains(node)) {
              if (compound) {
                const bool = this._matchLeaves(filterLeaves, node);
                if (bool) {
                  nodes.add(node);
                }
              } else {
                nodes.add(node);
              }
            }
          }
          break;
        }
        case SELECTOR_CLASS: {
          const arr = [].slice.call(baseNode.getElementsByClassName(leafName));
          if (arr.length) {
            if (compound) {
              for (const node of arr) {
                const bool = this._matchLeaves(filterLeaves, node);
                if (bool) {
                  nodes.add(node);
                }
              }
            } else {
              nodes = new Set(arr);
            }
          }
          break;
        }
        case SELECTOR_TYPE: {
          if (this.#document.contentType === 'text/html' &&
              !/[*|]/.test(leafName)) {
            const arr = [].slice.call(baseNode.getElementsByTagName(leafName));
            if (arr.length) {
              if (compound) {
                for (const node of arr) {
                  const bool = this._matchLeaves(filterLeaves, node);
                  if (bool) {
                    nodes.add(node);
                  }
                }
              } else {
                nodes = new Set(arr);
              }
            }
          } else {
            pending = true;
          }
          break;
        }
        case SELECTOR_PSEUDO_ELEMENT: {
          this._matchPseudoElementSelector(leafName);
          break;
        }
        default: {
          pending = true;
        }
      }
    }
    return {
      nodes,
      pending
    };
  }

  /**
   * match combinator
   * @param {object} twig - twig
   * @param {object} node - Element node
   * @param {object} [opt] - option
   * @param {string} [opt.dir] - direction to find
   * @param {boolean} [opt.forgive] - is forgiving selector list
   * @returns {Set.<object>} - collection of matched nodes
   */
  _matchCombinator(twig, node, opt = {}) {
    const { combo, leaves } = twig;
    const { name: comboName } = combo;
    const { dir, forgive } = opt;
    let matched = new Set();
    if (dir === DIR_NEXT) {
      switch (comboName) {
        case '+': {
          const refNode = node.nextElementSibling;
          if (refNode) {
            const bool = this._matchLeaves(leaves, refNode, { forgive });
            if (bool) {
              matched.add(refNode);
            }
          }
          break;
        }
        case '~': {
          const { parentNode } = node;
          if (parentNode) {
            const walker =
              this.#document.createTreeWalker(parentNode, SHOW_ELEMENT);
            let refNode = this._traverse(node, walker);
            if (refNode === node) {
              refNode = walker.nextSibling();
            }
            while (refNode) {
              const bool = this._matchLeaves(leaves, refNode, { forgive });
              if (bool) {
                matched.add(refNode);
              }
              refNode = walker.nextSibling();
            }
          }
          break;
        }
        case '>': {
          const walker = this.#document.createTreeWalker(node, SHOW_ELEMENT);
          let refNode = walker.firstChild();
          while (refNode) {
            const bool = this._matchLeaves(leaves, refNode, { forgive });
            if (bool) {
              matched.add(refNode);
            }
            refNode = walker.nextSibling();
          }
          break;
        }
        case ' ':
        default: {
          const { nodes, pending } = this._findDescendantNodes(leaves, node);
          if (nodes.size) {
            matched = nodes;
          } else if (pending) {
            const walker = this.#document.createTreeWalker(node, SHOW_ELEMENT);
            let refNode = walker.nextNode();
            while (refNode) {
              const bool = this._matchLeaves(leaves, refNode, { forgive });
              if (bool) {
                matched.add(refNode);
              }
              refNode = walker.nextNode();
            }
          }
        }
      }
    } else {
      switch (comboName) {
        case '+': {
          const refNode = node.previousElementSibling;
          if (refNode) {
            const bool = this._matchLeaves(leaves, refNode, { forgive });
            if (bool) {
              matched.add(refNode);
            }
          }
          break;
        }
        case '~': {
          const walker =
            this.#document.createTreeWalker(node.parentNode, SHOW_ELEMENT);
          let refNode = walker.firstChild();
          while (refNode) {
            if (refNode === node) {
              break;
            } else {
              const bool = this._matchLeaves(leaves, refNode, { forgive });
              if (bool) {
                matched.add(refNode);
              }
            }
            refNode = walker.nextSibling();
          }
          break;
        }
        case '>': {
          const refNode = node.parentNode;
          if (refNode) {
            const bool = this._matchLeaves(leaves, refNode, { forgive });
            if (bool) {
              matched.add(refNode);
            }
          }
          break;
        }
        case ' ':
        default: {
          const arr = [];
          let refNode = node.parentNode;
          while (refNode) {
            const bool = this._matchLeaves(leaves, refNode, { forgive });
            if (bool) {
              arr.push(refNode);
            }
            refNode = refNode.parentNode;
          }
          if (arr.length) {
            matched = new Set(arr.reverse());
          }
        }
      }
    }
    return matched;
  }

  /**
   * find matched node from tree walker
   * @param {Array.<object>} leaves - AST leaves
   * @param {object} [opt] - options
   * @param {object} [opt.node] - node to start from
   * @param {object} [opt.tree] - tree walker
   * @returns {?object} - matched node
   */
  _findNode(leaves, opt = {}) {
    let { node, walker } = opt;
    if (!walker) {
      walker = this.#tree;
    }
    let matchedNode;
    let refNode = this._traverse(node, walker);
    if (refNode) {
      if (refNode.nodeType !== ELEMENT_NODE) {
        refNode = walker.nextNode();
      } else if (refNode === node) {
        if (refNode !== this.#root) {
          refNode = walker.nextNode();
        }
      }
      while (refNode) {
        let bool;
        if (this.#node.nodeType === ELEMENT_NODE) {
          if (refNode === this.#node) {
            bool = true;
          } else {
            bool = this.#node.contains(refNode);
          }
        } else {
          bool = true;
        }
        if (bool) {
          const matched = this._matchLeaves(leaves, refNode);
          if (matched) {
            matchedNode = refNode;
            break;
          }
        }
        refNode = walker.nextNode();
      }
    }
    return matchedNode ?? null;
  }

  /**
   * find entry nodes
   * @param {object} twig - twig
   * @param {string} targetType - target type
   * @returns {object} - collection of nodes etc.
   */
  _findEntryNodes(twig, targetType) {
    const { leaves } = twig;
    const [leaf, ...filterLeaves] = leaves;
    const { type: leafType } = leaf;
    const leafName = unescapeSelector(leaf.name);
    const compound = filterLeaves.length > 0;
    let nodes = new Set();
    let filtered = false;
    let pending = false;
    switch (leafType) {
      case SELECTOR_ID: {
        if (targetType === TARGET_SELF) {
          const bool = this._matchLeaves(leaves, this.#node);
          if (bool) {
            nodes.add(this.#node);
            filtered = true;
          }
        } else if (targetType === TARGET_LINEAL) {
          let refNode = this.#node;
          while (refNode) {
            const bool = this._matchLeaves(leaves, refNode);
            if (bool) {
              nodes.add(refNode);
              filtered = true;
            }
            refNode = refNode.parentNode;
          }
        } else if (targetType === TARGET_ALL ||
                   this.#root.nodeType === ELEMENT_NODE) {
          pending = true;
        } else {
          const node = this.#root.getElementById(leafName);
          if (node) {
            nodes.add(node);
            filtered = true;
          }
        }
        break;
      }
      case SELECTOR_CLASS: {
        if (targetType === TARGET_SELF) {
          if (this.#node.nodeType === ELEMENT_NODE &&
              this.#node.classList.contains(leafName)) {
            nodes.add(this.#node);
          }
        } else if (targetType === TARGET_LINEAL) {
          let refNode = this.#node;
          while (refNode) {
            if (refNode.nodeType === ELEMENT_NODE) {
              if (refNode.classList.contains(leafName)) {
                nodes.add(refNode);
              }
              refNode = refNode.parentNode;
            } else {
              break;
            }
          }
        } else if (targetType === TARGET_FIRST) {
          const node = this._findNode(leaves, {
            node: this.#node,
            walker: this.#finder
          });
          if (node) {
            nodes.add(node);
            filtered = true;
            break;
          }
        } else if (this.#root.nodeType === DOCUMENT_FRAGMENT_NODE ||
                   this.#root.nodeType === ELEMENT_NODE) {
          pending = true;
        } else {
          const arr =
            [].slice.call(this.#root.getElementsByClassName(leafName));
          if (this.#node.nodeType === ELEMENT_NODE) {
            for (const node of arr) {
              if (node === this.#node || isInclusive(node, this.#node)) {
                nodes.add(node);
              }
            }
          } else if (arr.length) {
            nodes = new Set(arr);
          }
        }
        break;
      }
      case SELECTOR_TYPE: {
        if (targetType === TARGET_SELF) {
          if (this.#node.nodeType === ELEMENT_NODE) {
            const bool = this._matchLeaves(leaves, this.#node);
            if (bool) {
              nodes.add(this.#node);
              filtered = true;
            }
          }
        } else if (targetType === TARGET_LINEAL) {
          let refNode = this.#node;
          while (refNode) {
            if (refNode.nodeType === ELEMENT_NODE) {
              const bool = this._matchLeaves(leaves, refNode);
              if (bool) {
                nodes.add(refNode);
                filtered = true;
              }
              refNode = refNode.parentNode;
            } else {
              break;
            }
          }
        } else if (targetType === TARGET_FIRST) {
          const node = this._findNode(leaves, {
            node: this.#node,
            walker: this.#finder
          });
          if (node) {
            nodes.add(node);
            filtered = true;
            break;
          }
        } else if (this.#document.contentType !== 'text/html' ||
                   /[*|]/.test(leafName) ||
                   this.#root.nodeType === DOCUMENT_FRAGMENT_NODE ||
                   this.#root.nodeType === ELEMENT_NODE) {
          pending = true;
        } else {
          const arr = [].slice.call(this.#root.getElementsByTagName(leafName));
          if (this.#node.nodeType === ELEMENT_NODE) {
            for (const node of arr) {
              if (node === this.#node || isInclusive(node, this.#node)) {
                nodes.add(node);
              }
            }
          } else if (arr.length) {
            nodes = new Set(arr);
          }
        }
        break;
      }
      case SELECTOR_PSEUDO_ELEMENT: {
        // throws
        this._matchPseudoElementSelector(leafName);
        break;
      }
      default: {
        if (targetType !== TARGET_LINEAL && REG_SHADOW_HOST.test(leafName)) {
          if (this.#shadow && this.#node.nodeType === DOCUMENT_FRAGMENT_NODE) {
            const node = this._matchShadowHostPseudoClass(leaf, this.#node);
            if (node) {
              nodes.add(node);
            }
          }
        } else if (targetType === TARGET_SELF) {
          const bool = this._matchLeaves(leaves, this.#node);
          if (bool) {
            nodes.add(this.#node);
            filtered = true;
          }
        } else if (targetType === TARGET_LINEAL) {
          let refNode = this.#node;
          while (refNode) {
            const bool = this._matchLeaves(leaves, refNode);
            if (bool) {
              nodes.add(refNode);
              filtered = true;
            }
            refNode = refNode.parentNode;
          }
        } else if (targetType === TARGET_FIRST) {
          const node = this._findNode(leaves, {
            node: this.#node,
            walker: this.#finder
          });
          if (node) {
            nodes.add(node);
            filtered = true;
            break;
          }
        } else {
          pending = true;
        }
      }
    }
    return {
      compound,
      filtered,
      nodes,
      pending
    };
  }

  /**
   * get entry twig
   * @param {Array.<object>} branch - AST branch
   * @param {string} targetType - target type
   * @returns {object} - direction and twig
   */
  _getEntryTwig(branch, targetType) {
    const branchLen = branch.length;
    const complex = branchLen > 1;
    const firstTwig = branch[0];
    let dir;
    let twig;
    if (complex) {
      const { combo: firstCombo, leaves: [{ type: firstType }] } = firstTwig;
      const lastTwig = branch[branchLen - 1];
      const { leaves: [{ type: lastType }] } = lastTwig;
      if (lastType === SELECTOR_PSEUDO_ELEMENT || lastType === SELECTOR_ID) {
        dir = DIR_PREV;
        twig = lastTwig;
      } else if (firstType === SELECTOR_PSEUDO_ELEMENT ||
                 firstType === SELECTOR_ID) {
        dir = DIR_NEXT;
        twig = firstTwig;
      } else if (targetType === TARGET_ALL) {
        if (branchLen === 2) {
          const { name: comboName } = firstCombo;
          if (/^[+~]$/.test(comboName)) {
            dir = DIR_PREV;
            twig = lastTwig;
          } else {
            dir = DIR_NEXT;
            twig = firstTwig;
          }
        } else {
          dir = DIR_NEXT;
          twig = firstTwig;
        }
      } else {
        let bool;
        let sibling;
        for (const { combo, leaves: [leaf] } of branch) {
          const { type: leafType } = leaf;
          const leafName = unescapeSelector(leaf.name);
          if (leafType === SELECTOR_PSEUDO_CLASS && leafName === 'dir') {
            bool = false;
            break;
          }
          if (combo && !sibling) {
            const { name: comboName } = combo;
            if (/^[+~]$/.test(comboName)) {
              bool = true;
              sibling = true;
            }
          }
        }
        if (bool) {
          dir = DIR_NEXT;
          twig = firstTwig;
        } else {
          dir = DIR_PREV;
          twig = lastTwig;
        }
      }
    } else {
      dir = DIR_PREV;
      twig = firstTwig;
    }
    return {
      complex,
      dir,
      twig
    };
  }

  /**
   * collect nodes
   * @param {string} targetType - target type
   * @returns {Array.<Array.<object|undefined>>} - #ast and #nodes
   */
  _collectNodes(targetType) {
    const ast = this.#ast.values();
    if (targetType === TARGET_ALL || targetType === TARGET_FIRST) {
      const pendingItems = new Set();
      let i = 0;
      for (const { branch } of ast) {
        const { dir, twig } = this._getEntryTwig(branch, targetType);
        const {
          compound, filtered, nodes, pending
        } = this._findEntryNodes(twig, targetType);
        if (nodes.size) {
          this.#ast[i].find = true;
          this.#nodes[i] = nodes;
        } else if (pending) {
          pendingItems.add(new Map([
            ['index', i],
            ['twig', twig]
          ]));
        }
        this.#ast[i].dir = dir;
        this.#ast[i].filtered = filtered || !compound;
        i++;
      }
      if (pendingItems.size) {
        let node;
        let walker;
        if (this.#node !== this.#root && this.#node.nodeType === ELEMENT_NODE) {
          node = this.#node;
          walker = this.#finder;
        } else {
          node = this.#root;
          walker = this.#tree;
        }
        let nextNode = this._traverse(node, walker);
        while (nextNode) {
          let bool = false;
          if (this.#node.nodeType === ELEMENT_NODE) {
            if (nextNode === this.#node) {
              bool = true;
            } else {
              bool = this.#node.contains(nextNode);
            }
          } else {
            bool = true;
          }
          if (bool) {
            for (const pendingItem of pendingItems) {
              const { leaves } = pendingItem.get('twig');
              const matched = this._matchLeaves(leaves, nextNode);
              if (matched) {
                const index = pendingItem.get('index');
                this.#ast[index].filtered = true;
                this.#ast[index].find = true;
                this.#nodes[index].add(nextNode);
              }
            }
          }
          nextNode = walker.nextNode();
        }
      }
    } else {
      let i = 0;
      for (const { branch } of ast) {
        const twig = branch[branch.length - 1];
        const {
          compound, filtered, nodes
        } = this._findEntryNodes(twig, targetType);
        if (nodes.size) {
          this.#ast[i].find = true;
          this.#nodes[i] = nodes;
        }
        this.#ast[i].dir = DIR_PREV;
        this.#ast[i].filtered = filtered || !compound;
        i++;
      }
    }
    return [
      this.#ast,
      this.#nodes
    ];
  }

  /**
   * sort nodes
   * @param {Array.<object>|Set.<object>} nodes - collection of nodes
   * @returns {Array.<object|undefined>} - collection of sorted nodes
   */
  _sortNodes(nodes) {
    const arr = [...nodes];
    if (arr.length > 1) {
      arr.sort((a, b) => {
        let res;
        if (isPreceding(b, a)) {
          res = 1;
        } else {
          res = -1;
        }
        return res;
      });
    }
    return arr;
  }

  /**
   * match nodes
   * @param {string} targetType - target type
   * @returns {Set.<object>} - collection of matched nodes
   */
  _matchNodes(targetType) {
    const [...branches] = this.#ast;
    const l = branches.length;
    let nodes = new Set();
    for (let i = 0; i < l; i++) {
      const { branch, dir, filtered, find } = branches[i];
      const branchLen = branch.length;
      if (branchLen && find) {
        const entryNodes = this.#nodes[i];
        const lastIndex = branchLen - 1;
        if (lastIndex === 0) {
          const { leaves: [, ...filterLeaves] } = branch[0];
          if ((targetType === TARGET_ALL || targetType === TARGET_FIRST) &&
              this.#node.nodeType === ELEMENT_NODE) {
            for (const node of entryNodes) {
              const bool = filtered || this._matchLeaves(filterLeaves, node);
              if (bool && node !== this.#node && this.#node.contains(node)) {
                nodes.add(node);
                if (targetType !== TARGET_ALL) {
                  break;
                }
              }
            }
          } else if (!filterLeaves.length) {
            if (targetType === TARGET_ALL) {
              const n = [...nodes];
              nodes = new Set([...n, ...entryNodes]);
            } else {
              const [node] = [...entryNodes];
              nodes.add(node);
            }
          } else {
            for (const node of entryNodes) {
              const bool = filtered || this._matchLeaves(filterLeaves, node);
              if (bool) {
                nodes.add(node);
                if (targetType !== TARGET_ALL) {
                  break;
                }
              }
            }
          }
        } else if (dir === DIR_NEXT) {
          let { combo, leaves: entryLeaves } = branch[0];
          const [, ...filterLeaves] = entryLeaves;
          let matched;
          for (const node of entryNodes) {
            const bool = filtered || this._matchLeaves(filterLeaves, node);
            if (bool) {
              let nextNodes = new Set([node]);
              for (let j = 1; j < branchLen; j++) {
                const { combo: nextCombo, leaves } = branch[j];
                const arr = [];
                for (const nextNode of nextNodes) {
                  const twig = {
                    combo,
                    leaves
                  };
                  const m = this._matchCombinator(twig, nextNode, { dir });
                  if (m.size) {
                    arr.push(...m);
                  }
                }
                if (arr.length) {
                  if (j === lastIndex) {
                    if (targetType === TARGET_ALL) {
                      const n = [...nodes];
                      nodes = new Set([...n, ...arr]);
                    } else {
                      const [node] = this._sortNodes(arr);
                      nodes.add(node);
                    }
                    matched = true;
                  } else {
                    combo = nextCombo;
                    nextNodes = new Set(arr);
                    matched = false;
                  }
                } else {
                  matched = false;
                  break;
                }
              }
            } else {
              matched = false;
            }
            if (matched && targetType !== TARGET_ALL) {
              break;
            }
          }
          if (!matched && targetType === TARGET_FIRST) {
            const [entryNode] = [...entryNodes];
            let refNode = this._findNode(entryLeaves, {
              node: entryNode,
              walker: this.#finder
            });
            while (refNode) {
              let nextNodes = new Set([refNode]);
              for (let j = 1; j < branchLen; j++) {
                const { combo: nextCombo, leaves } = branch[j];
                const arr = [];
                for (const nextNode of nextNodes) {
                  const twig = {
                    combo,
                    leaves
                  };
                  const m = this._matchCombinator(twig, nextNode, { dir });
                  if (m.size) {
                    arr.push(...m);
                  }
                }
                if (arr.length) {
                  if (j === lastIndex) {
                    const [node] = this._sortNodes(arr);
                    nodes.add(node);
                    matched = true;
                  } else {
                    combo = nextCombo;
                    nextNodes = new Set(arr);
                    matched = false;
                  }
                } else {
                  matched = false;
                  break;
                }
              }
              if (matched) {
                break;
              }
              refNode = this._findNode(entryLeaves, {
                node: refNode,
                walker: this.#finder
              });
              nextNodes = new Set([refNode]);
            }
          }
        } else {
          const { leaves: entryLeaves } = branch[lastIndex];
          const [, ...filterLeaves] = entryLeaves;
          let matched;
          for (const node of entryNodes) {
            const bool = filtered || this._matchLeaves(filterLeaves, node);
            if (bool) {
              let nextNodes = new Set([node]);
              for (let j = lastIndex - 1; j >= 0; j--) {
                const twig = branch[j];
                const arr = [];
                for (const nextNode of nextNodes) {
                  const m = this._matchCombinator(twig, nextNode, { dir });
                  if (m.size) {
                    arr.push(...m);
                  }
                }
                if (arr.length) {
                  if (j === 0) {
                    nodes.add(node);
                    matched = true;
                  } else {
                    nextNodes = new Set(arr);
                    matched = false;
                  }
                } else {
                  matched = false;
                  break;
                }
              }
            }
            if (matched && targetType !== TARGET_ALL) {
              break;
            }
          }
          if (!matched && targetType === TARGET_FIRST) {
            const [entryNode] = [...entryNodes];
            let refNode = this._findNode(entryLeaves, {
              node: entryNode,
              walker: this.#finder
            });
            while (refNode) {
              let nextNodes = new Set([refNode]);
              for (let j = lastIndex - 1; j >= 0; j--) {
                const twig = branch[j];
                const arr = [];
                for (const nextNode of nextNodes) {
                  const m = this._matchCombinator(twig, nextNode, { dir });
                  if (m.size) {
                    arr.push(...m);
                  }
                }
                if (arr.length) {
                  if (j === 0) {
                    nodes.add(refNode);
                    matched = true;
                  } else {
                    nextNodes = new Set(arr);
                    matched = false;
                  }
                } else {
                  matched = false;
                  break;
                }
              }
              if (matched) {
                break;
              }
              refNode = this._findNode(entryLeaves, {
                node: refNode,
                walker: this.#finder
              });
              nextNodes = new Set([refNode]);
            }
          }
        }
      }
    }
    return nodes;
  }

  /**
   * find matched nodes
   * @param {string} targetType - target type
   * @returns {Set.<object>} - collection of matched nodes
   */
  _find(targetType) {
    if (targetType === TARGET_ALL || targetType === TARGET_FIRST) {
      this.#finder = this.#document.createTreeWalker(this.#node, SHOW_ELEMENT);
    }
    this._collectNodes(targetType);
    const nodes = this._matchNodes(targetType);
    return nodes;
  }

  /**
   * matches
   * @returns {boolean} - `true` if matched `false` otherwise
   */
  matches() {
    if (this.#node.nodeType !== ELEMENT_NODE) {
      const msg = `Unexpected node ${this.#node.nodeName}`;
      this._onError(new TypeError(msg));
    }
    let res;
    try {
      const nodes = this._find(TARGET_SELF);
      if (nodes.size) {
        res = nodes.has(this.#node);
      }
    } catch (e) {
      this._onError(e);
    }
    return !!res;
  }

  /**
   * closest
   * @returns {?object} - matched node
   */
  closest() {
    if (this.#node.nodeType !== ELEMENT_NODE) {
      const msg = `Unexpected node ${this.#node.nodeName}`;
      this._onError(new TypeError(msg));
    }
    let res;
    try {
      const nodes = this._find(TARGET_LINEAL);
      let node = this.#node;
      while (node) {
        if (nodes.has(node)) {
          res = node;
          break;
        }
        node = node.parentNode;
      }
    } catch (e) {
      this._onError(e);
    }
    return res ?? null;
  }

  /**
   * query selector
   * @returns {?object} - matched node
   */
  querySelector() {
    let res;
    try {
      const nodes = this._find(TARGET_FIRST);
      nodes.delete(this.#node);
      if (nodes.size) {
        [res] = this._sortNodes(nodes);
      }
    } catch (e) {
      this._onError(e);
    }
    return res ?? null;
  }

  /**
   * query selector all
   * NOTE: returns Array, not NodeList
   * @returns {Array.<object|undefined>} - collection of matched nodes
   */
  querySelectorAll() {
    let res;
    try {
      const nodes = this._find(TARGET_ALL);
      nodes.delete(this.#node);
      if (nodes.size) {
        res = this._sortNodes(nodes);
      }
    } catch (e) {
      this._onError(e);
    }
    return res ?? [];
  }
};
