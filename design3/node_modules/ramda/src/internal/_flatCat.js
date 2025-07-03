var _forceReduced =
/*#__PURE__*/
require("./_forceReduced.js");

var _isArrayLike =
/*#__PURE__*/
require("./_isArrayLike.js");

var _xArrayReduce =
/*#__PURE__*/
require("./_xArrayReduce.js");

var _xReduce =
/*#__PURE__*/
require("./_xReduce.js");

var _xfBase =
/*#__PURE__*/
require("./_xfBase.js");

var tInit = '@@transducer/init';
var tStep = '@@transducer/step';
var tResult = '@@transducer/result';

var XPreservingReduced =
/*#__PURE__*/
function () {
  function XPreservingReduced(xf) {
    this.xf = xf;
  }

  XPreservingReduced.prototype[tInit] = _xfBase.init;
  XPreservingReduced.prototype[tResult] = _xfBase.result;

  XPreservingReduced.prototype[tStep] = function (result, input) {
    var ret = this.xf[tStep](result, input);
    return ret['@@transducer/reduced'] ? _forceReduced(ret) : ret;
  };

  return XPreservingReduced;
}();

var XFlatCat =
/*#__PURE__*/
function () {
  function XFlatCat(xf) {
    this.xf = new XPreservingReduced(xf);
  }

  XFlatCat.prototype[tInit] = _xfBase.init;
  XFlatCat.prototype[tResult] = _xfBase.result;

  XFlatCat.prototype[tStep] = function (result, input) {
    return !_isArrayLike(input) ? _xArrayReduce(this.xf, result, [input]) : _xReduce(this.xf, result, input);
  };

  return XFlatCat;
}();

var _flatCat = function _xcat(xf) {
  return new XFlatCat(xf);
};

module.exports = _flatCat;