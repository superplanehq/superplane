import _curry3 from "./_curry3.js";
import _xfBase from "./_xfBase.js";
var tInit = '@@transducer/init';
var tStep = '@@transducer/step';

var XScan =
/*#__PURE__*/
function () {
  function XScan(reducer, acc, xf) {
    this.xf = xf;
    this.f = reducer;
    this.acc = acc;
  }

  XScan.prototype[tInit] = function () {
    return this.xf[tStep](this.xf[tInit](), this.acc);
  };

  XScan.prototype['@@transducer/result'] = _xfBase.result;

  XScan.prototype[tStep] = function (result, input) {
    if (result['@@transducer/reduced']) {
      return result;
    }

    this.acc = this.f(this.acc, input);
    return this.xf[tStep](result, this.acc);
  };

  return XScan;
}();

var _xscan =
/*#__PURE__*/
_curry3(function _xscan(reducer, acc, xf) {
  return new XScan(reducer, acc, xf);
});

export default _xscan;