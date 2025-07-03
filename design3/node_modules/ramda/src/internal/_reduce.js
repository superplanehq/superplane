var _arrayReduce =
/*#__PURE__*/
require("./_arrayReduce.js");

var _createReduce =
/*#__PURE__*/
require("./_createReduce.js");

function _iterableReduce(reducer, acc, iter) {
  var step = iter.next();

  while (!step.done) {
    acc = reducer(acc, step.value);
    step = iter.next();
  }

  return acc;
}

function _methodReduce(reducer, acc, obj, methodName) {
  return obj[methodName](reducer, acc);
}

var _reduce =
/*#__PURE__*/
_createReduce(_arrayReduce, _methodReduce, _iterableReduce);

module.exports = _reduce;