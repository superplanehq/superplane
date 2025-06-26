var _bindCb = require('./_bindCb.js');
var _setup = require('./_setup.js');

function unmethodize(method) {
  return _bindCb(_setup.call, method);
}

module.exports = unmethodize;
