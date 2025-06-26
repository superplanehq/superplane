define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var splice = _unmethodize(_setup.ArrayProto.splice);

	return splice;

});
