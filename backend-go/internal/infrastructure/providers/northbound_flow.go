package providers

import marketdomain "stock-predict-go/internal/domain/market"

func hasMeaningfulNorthboundFlow(flow *marketdomain.NorthboundFlow) bool {
	if flow == nil {
		return false
	}
	if flow.SHNetBuy != 0 || flow.SZNetBuy != 0 || flow.TotalBuy != 0 {
		return true
	}
	for _, point := range flow.Timeline {
		if point.SHFlow != 0 || point.SZFlow != 0 {
			return true
		}
	}
	return false
}
