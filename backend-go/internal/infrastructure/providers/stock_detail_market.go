package providers

func marketToSecID(market string) int {
	switch market {
	case "sh":
		return 1
	case "sz":
		return 0
	case "bj":
		return 0
	default:
		return 1
	}
}
