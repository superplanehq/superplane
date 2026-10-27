package billinghttp

import (
	"net/http"

	"github.com/superplanehq/superplane/ee/billing/polar"
)

type adminPolarProduct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Server) adminListPolarProducts(w http.ResponseWriter, r *http.Request) {
	kind := polar.ProductKind(r.URL.Query().Get("kind"))
	switch kind {
	case "":
		kind = polar.ProductKindOneTime
	case polar.ProductKindOneTime, polar.ProductKindSubscription:
	default:
		http.Error(w, "Invalid Polar product kind.", http.StatusBadRequest)
		return
	}
	if !polar.Configured() {
		http.Error(w, "Configure Polar to load products.", http.StatusServiceUnavailable)
		return
	}
	products, err := polar.NewClientFromEnv().ListProducts(r.Context(), kind)
	if err != nil {
		writePolarProductError(w, err)
		return
	}
	items := make([]adminPolarProduct, 0, len(products))
	for _, product := range products {
		items = append(items, adminPolarProduct{ID: product.ID, Name: product.Name})
	}
	respondJSON(w, items)
}

func writePolarProductError(w http.ResponseWriter, err error) {
	if polar.IsUnauthorized(err) {
		http.Error(w, "Could not load Polar products. Check the access token and product read permission.", http.StatusBadGateway)
		return
	}
	if polar.IsRateLimited(err) {
		http.Error(w, "Polar is busy. Try again shortly.", http.StatusTooManyRequests)
		return
	}
	http.Error(w, "Could not load Polar products. Try again.", http.StatusBadGateway)
}
