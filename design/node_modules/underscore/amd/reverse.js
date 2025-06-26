define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var reverse = _unmethodize(_setup.ArrayProto.reverse);

	return reverse;

});
