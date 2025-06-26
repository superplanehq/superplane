define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var shift = _unmethodize(_setup.ArrayProto.shift);

	return shift;

});
