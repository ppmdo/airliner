package calculation

import "airliner/model"

func GetMinPriceOffer(offers []*model.Offer) *model.Offer {
	var min *model.Offer

	for _, o := range offers {

		if min == nil || o.Price < min.Price {
			min = o
		}
	}

	return min
}
