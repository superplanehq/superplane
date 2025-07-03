'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var isPlainObject = require('is-plain-object');
var React = require('react');
var prettyPrintObject = require('@base2/pretty-print-object');
var reactIs = require('react-is');

function _interopDefaultLegacy (e) { return e && typeof e === 'object' && 'default' in e ? e : { 'default': e }; }

function _interopNamespace(e) {
  if (e && e.__esModule) return e;
  var n = Object.create(null);
  if (e) {
    Object.keys(e).forEach(function (k) {
      if (k !== 'default') {
        var d = Object.getOwnPropertyDescriptor(e, k);
        Object.defineProperty(n, k, d.get ? d : {
          enumerable: true,
          get: function () { return e[k]; }
        });
      }
    });
  }
  n["default"] = e;
  return Object.freeze(n);
}

var React__namespace = /*#__PURE__*/_interopNamespace(React);
var React__default = /*#__PURE__*/_interopDefaultLegacy(React);

var spacer = (function (times, tabStop) {
  if (times === 0) {
    return '';
  }

  return new Array(times * tabStop).fill(' ').join('');
});

function _typeof(obj) {
  "@babel/helpers - typeof";

  return _typeof = "function" == typeof Symbol && "symbol" == typeof Symbol.iterator ? function (obj) {
    return typeof obj;
  } : function (obj) {
    return obj && "function" == typeof Symbol && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj;
  }, _typeof(obj);
}

function _toConsumableArray(arr) {
  return _arrayWithoutHoles(arr) || _iterableToArray(arr) || _unsupportedIterableToArray(arr) || _nonIterableSpread();
}

function _arrayWithoutHoles(arr) {
  if (Array.isArray(arr)) return _arrayLikeToArray(arr);
}

function _iterableToArray(iter) {
  if (typeof Symbol !== "undefined" && iter[Symbol.iterator] != null || iter["@@iterator"] != null) return Array.from(iter);
}

function _unsupportedIterableToArray(o, minLen) {
  if (!o) return;
  if (typeof o === "string") return _arrayLikeToArray(o, minLen);
  var n = Object.prototype.toString.call(o).slice(8, -1);
  if (n === "Object" && o.constructor) n = o.constructor.name;
  if (n === "Map" || n === "Set") return Array.from(o);
  if (n === "Arguments" || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(n)) return _arrayLikeToArray(o, minLen);
}

function _arrayLikeToArray(arr, len) {
  if (len == null || len > arr.length) len = arr.length;

  for (var i = 0, arr2 = new Array(len); i < len; i++) arr2[i] = arr[i];

  return arr2;
}

function _nonIterableSpread() {
  throw new TypeError("Invalid attempt to spread non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method.");
}

function safeSortObject(value, seen) {
  // return non-object value as is
  if (value === null || _typeof(value) !== 'object') {
    return value;
  } // return date, regexp and react element values as is


  if (value instanceof Date || value instanceof RegExp || /*#__PURE__*/React__namespace.isValidElement(value)) {
    return value;
  }

  seen.add(value); // make a copy of array with each item passed through the sorting algorithm

  if (Array.isArray(value)) {
    return value.map(function (v) {
      return safeSortObject(v, seen);
    });
  } // make a copy of object with key sorted


  return Object.keys(value).sort().reduce(function (result, key) {
    if (key === '_owner') {
      return result;
    }

    if (key === 'current' || seen.has(value[key])) {
      // eslint-disable-next-line no-param-reassign
      result[key] = '[Circular]';
    } else {
      // eslint-disable-next-line no-param-reassign
      result[key] = safeSortObject(value[key], seen);
    }

    return result;
  }, {});
}

function sortObject(value) {
  return safeSortObject(value, new WeakSet());
}

/* eslint-disable no-use-before-define */
var createStringTreeNode = function createStringTreeNode(value) {
  return {
    type: 'string',
    value: value
  };
};
var createNumberTreeNode = function createNumberTreeNode(value) {
  return {
    type: 'number',
    value: value
  };
};
var createReactElementTreeNode = function createReactElementTreeNode(displayName, props, defaultProps, childrens) {
  return {
    type: 'ReactElement',
    displayName: displayName,
    props: props,
    defaultProps: defaultProps,
    childrens: childrens
  };
};
var createReactFragmentTreeNode = function createReactFragmentTreeNode(key, childrens) {
  return {
    type: 'ReactFragment',
    key: key,
    childrens: childrens
  };
};

var supportFragment = Boolean(React.Fragment);

var getFunctionTypeName = function getFunctionTypeName(functionType) {
  if (!functionType.name || functionType.name === '_default') {
    return 'No Display Name';
  }

  return functionType.name;
};

var getWrappedComponentDisplayName = function getWrappedComponentDisplayName(Component) {
  switch (true) {
    case Boolean(Component.displayName):
      return Component.displayName;

    case Component.$$typeof === reactIs.Memo:
      return getWrappedComponentDisplayName(Component.type);

    case Component.$$typeof === reactIs.ForwardRef:
      return getWrappedComponentDisplayName(Component.render);

    default:
      return getFunctionTypeName(Component);
  }
}; // heavily inspired by:
// https://github.com/facebook/react/blob/3746eaf985dd92f8aa5f5658941d07b6b855e9d9/packages/react-devtools-shared/src/backend/renderer.js#L399-L496


var getReactElementDisplayName = function getReactElementDisplayName(element) {
  switch (true) {
    case typeof element.type === 'string':
      return element.type;

    case typeof element.type === 'function':
      if (element.type.displayName) {
        return element.type.displayName;
      }

      return getFunctionTypeName(element.type);

    case reactIs.isForwardRef(element):
    case reactIs.isMemo(element):
      return getWrappedComponentDisplayName(element.type);

    case reactIs.isContextConsumer(element):
      return "".concat(element.type._context.displayName || 'Context', ".Consumer");

    case reactIs.isContextProvider(element):
      return "".concat(element.type._context.displayName || 'Context', ".Provider");

    case reactIs.isLazy(element):
      return 'Lazy';

    case reactIs.isProfiler(element):
      return 'Profiler';

    case reactIs.isStrictMode(element):
      return 'StrictMode';

    case reactIs.isSuspense(element):
      return 'Suspense';

    default:
      return 'UnknownElementType';
  }
};

var noChildren = function noChildren(propsValue, propName) {
  return propName !== 'children';
};

var onlyMeaningfulChildren = function onlyMeaningfulChildren(children) {
  return children !== true && children !== false && children !== null && children !== '';
};

var filterProps = function filterProps(originalProps, cb) {
  var filteredProps = {};
  Object.keys(originalProps).filter(function (key) {
    return cb(originalProps[key], key);
  }).forEach(function (key) {
    return filteredProps[key] = originalProps[key];
  });
  return filteredProps;
};

var parseReactElement = function parseReactElement(element, options) {
  var _options$displayName = options.displayName,
      displayNameFn = _options$displayName === void 0 ? getReactElementDisplayName : _options$displayName;

  if (typeof element === 'string') {
    return createStringTreeNode(element);
  } else if (typeof element === 'number') {
    return createNumberTreeNode(element);
  } else if (! /*#__PURE__*/React__default["default"].isValidElement(element)) {
    throw new Error("react-element-to-jsx-string: Expected a React.Element, got `".concat(_typeof(element), "`"));
  }

  var displayName = displayNameFn(element);
  var props = filterProps(element.props, noChildren);

  if (element.ref !== null) {
    props.ref = element.ref;
  }

  var key = element.key;

  if (typeof key === 'string' && key.search(/^\./)) {
    // React automatically add key=".X" when there are some children
    props.key = key;
  }

  var defaultProps = filterProps(element.type.defaultProps || {}, noChildren);
  var childrens = React__default["default"].Children.toArray(element.props.children).filter(onlyMeaningfulChildren).map(function (child) {
    return parseReactElement(child, options);
  });

  if (supportFragment && element.type === React.Fragment) {
    return createReactFragmentTreeNode(key, childrens);
  }

  return createReactElementTreeNode(displayName, props, defaultProps, childrens);
};

function noRefCheck() {}

var inlineFunction = function inlineFunction(fn) {
  return fn.toString().split('\n').map(function (line) {
    return line.trim();
  }).join('');
};
var preserveFunctionLineBreak = function preserveFunctionLineBreak(fn) {
  return fn.toString();
};
var defaultFunctionValue = inlineFunction;
var formatFunction = (function (fn, options) {
  var _options$functionValu = options.functionValue,
      functionValue = _options$functionValu === void 0 ? defaultFunctionValue : _options$functionValu,
      showFunctions = options.showFunctions;

  if (!showFunctions && functionValue === defaultFunctionValue) {
    return functionValue(noRefCheck);
  }

  return functionValue(fn);
});

var formatComplexDataStructure = (function (value, inline, lvl, options) {
  var normalizedValue = sortObject(value);
  var stringifiedValue = prettyPrintObject.prettyPrint(normalizedValue, {
    transform: function transform(currentObj, prop, originalResult) {
      var currentValue = currentObj[prop];

      if (currentValue && /*#__PURE__*/React.isValidElement(currentValue)) {
        return formatTreeNode(parseReactElement(currentValue, options), true, lvl, options);
      }

      if (typeof currentValue === 'function') {
        return formatFunction(currentValue, options);
      }

      return originalResult;
    }
  });

  if (inline) {
    return stringifiedValue.replace(/\s+/g, ' ').replace(/{ /g, '{').replace(/ }/g, '}').replace(/\[ /g, '[').replace(/ ]/g, ']');
  } // Replace tabs with spaces, and add necessary indentation in front of each new line


  return stringifiedValue.replace(/\t/g, spacer(1, options.tabStop)).replace(/\n([^$])/g, "\n".concat(spacer(lvl + 1, options.tabStop), "$1"));
});

var escape$1 = function escape(s) {
  return s.replace(/"/g, '&quot;');
};

var formatPropValue = function formatPropValue(propValue, inline, lvl, options) {
  if (typeof propValue === 'number') {
    return "{".concat(String(propValue), "}");
  }

  if (typeof propValue === 'string') {
    return "\"".concat(escape$1(propValue), "\"");
  } // > "Symbols (new in ECMAScript 2015, not yet supported in Flow)"
  // @see: https://flow.org/en/docs/types/primitives/
  // $FlowFixMe: Flow does not support Symbol


  if (_typeof(propValue) === 'symbol') {
    var symbolDescription = propValue.valueOf().toString().replace(/Symbol\((.*)\)/, '$1');

    if (!symbolDescription) {
      return "{Symbol()}";
    }

    return "{Symbol('".concat(symbolDescription, "')}");
  }

  if (typeof propValue === 'function') {
    return "{".concat(formatFunction(propValue, options), "}");
  }

  if ( /*#__PURE__*/React.isValidElement(propValue)) {
    return "{".concat(formatTreeNode(parseReactElement(propValue, options), true, lvl, options), "}");
  }

  if (propValue instanceof Date) {
    if (isNaN(propValue.valueOf())) {
      return "{new Date(NaN)}";
    }

    return "{new Date(\"".concat(propValue.toISOString(), "\")}");
  }

  if (isPlainObject.isPlainObject(propValue) || Array.isArray(propValue)) {
    return "{".concat(formatComplexDataStructure(propValue, inline, lvl, options), "}");
  }

  return "{".concat(String(propValue), "}");
};

var formatProp = (function (name, hasValue, value, hasDefaultValue, defaultValue, inline, lvl, options) {
  if (!hasValue && !hasDefaultValue) {
    throw new Error("The prop \"".concat(name, "\" has no value and no default: could not be formatted"));
  }

  var usedValue = hasValue ? value : defaultValue;
  var useBooleanShorthandSyntax = options.useBooleanShorthandSyntax,
      tabStop = options.tabStop;
  var formattedPropValue = formatPropValue(usedValue, inline, lvl, options);
  var attributeFormattedInline = ' ';
  var attributeFormattedMultiline = "\n".concat(spacer(lvl + 1, tabStop));
  var isMultilineAttribute = formattedPropValue.includes('\n');

  if (useBooleanShorthandSyntax && formattedPropValue === '{false}' && !hasDefaultValue) {
    // If a boolean is false and not different from it's default, we do not render the attribute
    attributeFormattedInline = '';
    attributeFormattedMultiline = '';
  } else if (useBooleanShorthandSyntax && formattedPropValue === '{true}') {
    attributeFormattedInline += "".concat(name);
    attributeFormattedMultiline += "".concat(name);
  } else {
    attributeFormattedInline += "".concat(name, "=").concat(formattedPropValue);
    attributeFormattedMultiline += "".concat(name, "=").concat(formattedPropValue);
  }

  return {
    attributeFormattedInline: attributeFormattedInline,
    attributeFormattedMultiline: attributeFormattedMultiline,
    isMultilineAttribute: isMultilineAttribute
  };
});

var mergeSiblingPlainStringChildrenReducer = (function (previousNodes, currentNode) {
  var nodes = previousNodes.slice(0, previousNodes.length > 0 ? previousNodes.length - 1 : 0);
  var previousNode = previousNodes[previousNodes.length - 1];

  if (previousNode && (currentNode.type === 'string' || currentNode.type === 'number') && (previousNode.type === 'string' || previousNode.type === 'number')) {
    nodes.push(createStringTreeNode(String(previousNode.value) + String(currentNode.value)));
  } else {
    if (previousNode) {
      nodes.push(previousNode);
    }

    nodes.push(currentNode);
  }

  return nodes;
});

var isKeyOrRefProps = function isKeyOrRefProps(propName) {
  return ['key', 'ref'].includes(propName);
};

var sortPropsByNames = (function (shouldSortUserProps) {
  return function (props) {
    var haveKeyProp = props.includes('key');
    var haveRefProp = props.includes('ref');
    var userPropsOnly = props.filter(function (oneProp) {
      return !isKeyOrRefProps(oneProp);
    });
    var sortedProps = shouldSortUserProps ? _toConsumableArray(userPropsOnly.sort()) // We use basic lexical order
    : _toConsumableArray(userPropsOnly);

    if (haveRefProp) {
      sortedProps.unshift('ref');
    }

    if (haveKeyProp) {
      sortedProps.unshift('key');
    }

    return sortedProps;
  };
});

function createPropFilter(props, filter) {
  if (Array.isArray(filter)) {
    return function (key) {
      return filter.indexOf(key) === -1;
    };
  } else {
    return function (key) {
      return filter(props[key], key);
    };
  }
}

var compensateMultilineStringElementIndentation = function compensateMultilineStringElementIndentation(element, formattedElement, inline, lvl, options) {
  var tabStop = options.tabStop;

  if (element.type === 'string') {
    return formattedElement.split('\n').map(function (line, offset) {
      if (offset === 0) {
        return line;
      }

      return "".concat(spacer(lvl, tabStop)).concat(line);
    }).join('\n');
  }

  return formattedElement;
};

var formatOneChildren = function formatOneChildren(inline, lvl, options) {
  return function (element) {
    return compensateMultilineStringElementIndentation(element, formatTreeNode(element, inline, lvl, options), inline, lvl, options);
  };
};

var onlyPropsWithOriginalValue = function onlyPropsWithOriginalValue(defaultProps, props) {
  return function (propName) {
    var haveDefaultValue = Object.keys(defaultProps).includes(propName);
    return !haveDefaultValue || haveDefaultValue && defaultProps[propName] !== props[propName];
  };
};

var isInlineAttributeTooLong = function isInlineAttributeTooLong(attributes, inlineAttributeString, lvl, tabStop, maxInlineAttributesLineLength) {
  if (!maxInlineAttributesLineLength) {
    return attributes.length > 1;
  }

  return spacer(lvl, tabStop).length + inlineAttributeString.length > maxInlineAttributesLineLength;
};

var shouldRenderMultilineAttr = function shouldRenderMultilineAttr(attributes, inlineAttributeString, containsMultilineAttr, inline, lvl, tabStop, maxInlineAttributesLineLength) {
  return (isInlineAttributeTooLong(attributes, inlineAttributeString, lvl, tabStop, maxInlineAttributesLineLength) || containsMultilineAttr) && !inline;
};

var formatReactElementNode = (function (node, inline, lvl, options) {
  var type = node.type,
      _node$displayName = node.displayName,
      displayName = _node$displayName === void 0 ? '' : _node$displayName,
      childrens = node.childrens,
      _node$props = node.props,
      props = _node$props === void 0 ? {} : _node$props,
      _node$defaultProps = node.defaultProps,
      defaultProps = _node$defaultProps === void 0 ? {} : _node$defaultProps;

  if (type !== 'ReactElement') {
    throw new Error("The \"formatReactElementNode\" function could only format node of type \"ReactElement\". Given:  ".concat(type));
  }

  var filterProps = options.filterProps,
      maxInlineAttributesLineLength = options.maxInlineAttributesLineLength,
      showDefaultProps = options.showDefaultProps,
      sortProps = options.sortProps,
      tabStop = options.tabStop;
  var out = "<".concat(displayName);
  var outInlineAttr = out;
  var outMultilineAttr = out;
  var containsMultilineAttr = false;
  var visibleAttributeNames = [];
  var propFilter = createPropFilter(props, filterProps);
  Object.keys(props).filter(propFilter).filter(onlyPropsWithOriginalValue(defaultProps, props)).forEach(function (propName) {
    return visibleAttributeNames.push(propName);
  });
  Object.keys(defaultProps).filter(propFilter).filter(function () {
    return showDefaultProps;
  }).filter(function (defaultPropName) {
    return !visibleAttributeNames.includes(defaultPropName);
  }).forEach(function (defaultPropName) {
    return visibleAttributeNames.push(defaultPropName);
  });
  var attributes = sortPropsByNames(sortProps)(visibleAttributeNames);
  attributes.forEach(function (attributeName) {
    var _formatProp = formatProp(attributeName, Object.keys(props).includes(attributeName), props[attributeName], Object.keys(defaultProps).includes(attributeName), defaultProps[attributeName], inline, lvl, options),
        attributeFormattedInline = _formatProp.attributeFormattedInline,
        attributeFormattedMultiline = _formatProp.attributeFormattedMultiline,
        isMultilineAttribute = _formatProp.isMultilineAttribute;

    if (isMultilineAttribute) {
      containsMultilineAttr = true;
    }

    outInlineAttr += attributeFormattedInline;
    outMultilineAttr += attributeFormattedMultiline;
  });
  outMultilineAttr += "\n".concat(spacer(lvl, tabStop));

  if (shouldRenderMultilineAttr(attributes, outInlineAttr, containsMultilineAttr, inline, lvl, tabStop, maxInlineAttributesLineLength)) {
    out = outMultilineAttr;
  } else {
    out = outInlineAttr;
  }

  if (childrens && childrens.length > 0) {
    var newLvl = lvl + 1;
    out += '>';

    if (!inline) {
      out += '\n';
      out += spacer(newLvl, tabStop);
    }

    out += childrens.reduce(mergeSiblingPlainStringChildrenReducer, []).map(formatOneChildren(inline, newLvl, options)).join(!inline ? "\n".concat(spacer(newLvl, tabStop)) : '');

    if (!inline) {
      out += '\n';
      out += spacer(newLvl - 1, tabStop);
    }

    out += "</".concat(displayName, ">");
  } else {
    if (!isInlineAttributeTooLong(attributes, outInlineAttr, lvl, tabStop, maxInlineAttributesLineLength)) {
      out += ' ';
    }

    out += '/>';
  }

  return out;
});

var REACT_FRAGMENT_TAG_NAME_SHORT_SYNTAX = '';
var REACT_FRAGMENT_TAG_NAME_EXPLICIT_SYNTAX = 'React.Fragment';

var toReactElementTreeNode = function toReactElementTreeNode(displayName, key, childrens) {
  var props = {};

  if (key) {
    props = {
      key: key
    };
  }

  return {
    type: 'ReactElement',
    displayName: displayName,
    props: props,
    defaultProps: {},
    childrens: childrens
  };
};

var isKeyedFragment = function isKeyedFragment(_ref) {
  var key = _ref.key;
  return Boolean(key);
};

var hasNoChildren = function hasNoChildren(_ref2) {
  var childrens = _ref2.childrens;
  return childrens.length === 0;
};

var formatReactFragmentNode = (function (node, inline, lvl, options) {
  var type = node.type,
      key = node.key,
      childrens = node.childrens;

  if (type !== 'ReactFragment') {
    throw new Error("The \"formatReactFragmentNode\" function could only format node of type \"ReactFragment\". Given: ".concat(type));
  }

  var useFragmentShortSyntax = options.useFragmentShortSyntax;
  var displayName;

  if (useFragmentShortSyntax) {
    if (hasNoChildren(node) || isKeyedFragment(node)) {
      displayName = REACT_FRAGMENT_TAG_NAME_EXPLICIT_SYNTAX;
    } else {
      displayName = REACT_FRAGMENT_TAG_NAME_SHORT_SYNTAX;
    }
  } else {
    displayName = REACT_FRAGMENT_TAG_NAME_EXPLICIT_SYNTAX;
  }

  return formatReactElementNode(toReactElementTreeNode(displayName, key, childrens), inline, lvl, options);
});

var jsxStopChars = ['<', '>', '{', '}'];

var shouldBeEscaped = function shouldBeEscaped(s) {
  return jsxStopChars.some(function (jsxStopChar) {
    return s.includes(jsxStopChar);
  });
};

var escape = function escape(s) {
  if (!shouldBeEscaped(s)) {
    return s;
  }

  return "{`".concat(s, "`}");
};

var preserveTrailingSpace = function preserveTrailingSpace(s) {
  var result = s;

  if (result.endsWith(' ')) {
    result = result.replace(/^(.*?)(\s+)$/, "$1{'$2'}");
  }

  if (result.startsWith(' ')) {
    result = result.replace(/^(\s+)(.*)$/, "{'$1'}$2");
  }

  return result;
};

var formatTreeNode = (function (node, inline, lvl, options) {
  if (node.type === 'number') {
    return String(node.value);
  }

  if (node.type === 'string') {
    return node.value ? "".concat(preserveTrailingSpace(escape(String(node.value)))) : '';
  }

  if (node.type === 'ReactElement') {
    return formatReactElementNode(node, inline, lvl, options);
  }

  if (node.type === 'ReactFragment') {
    return formatReactFragmentNode(node, inline, lvl, options);
  }

  throw new TypeError("Unknow format type \"".concat(node.type, "\""));
});

var formatTree = (function (node, options) {
  return formatTreeNode(node, false, 0, options);
});

var reactElementToJsxString = function reactElementToJsxString(element) {
  var _ref = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : {},
      _ref$filterProps = _ref.filterProps,
      filterProps = _ref$filterProps === void 0 ? [] : _ref$filterProps,
      _ref$showDefaultProps = _ref.showDefaultProps,
      showDefaultProps = _ref$showDefaultProps === void 0 ? true : _ref$showDefaultProps,
      _ref$showFunctions = _ref.showFunctions,
      showFunctions = _ref$showFunctions === void 0 ? false : _ref$showFunctions,
      functionValue = _ref.functionValue,
      _ref$tabStop = _ref.tabStop,
      tabStop = _ref$tabStop === void 0 ? 2 : _ref$tabStop,
      _ref$useBooleanShorth = _ref.useBooleanShorthandSyntax,
      useBooleanShorthandSyntax = _ref$useBooleanShorth === void 0 ? true : _ref$useBooleanShorth,
      _ref$useFragmentShort = _ref.useFragmentShortSyntax,
      useFragmentShortSyntax = _ref$useFragmentShort === void 0 ? true : _ref$useFragmentShort,
      _ref$sortProps = _ref.sortProps,
      sortProps = _ref$sortProps === void 0 ? true : _ref$sortProps,
      maxInlineAttributesLineLength = _ref.maxInlineAttributesLineLength,
      displayName = _ref.displayName;

  if (!element) {
    throw new Error('react-element-to-jsx-string: Expected a ReactElement');
  }

  var options = {
    filterProps: filterProps,
    showDefaultProps: showDefaultProps,
    showFunctions: showFunctions,
    functionValue: functionValue,
    tabStop: tabStop,
    useBooleanShorthandSyntax: useBooleanShorthandSyntax,
    useFragmentShortSyntax: useFragmentShortSyntax,
    sortProps: sortProps,
    maxInlineAttributesLineLength: maxInlineAttributesLineLength,
    displayName: displayName
  };
  return formatTree(parseReactElement(element, options), options);
};

exports["default"] = reactElementToJsxString;
exports.inlineFunction = inlineFunction;
exports.preserveFunctionLineBreak = preserveFunctionLineBreak;
//# sourceMappingURL=index.js.map
