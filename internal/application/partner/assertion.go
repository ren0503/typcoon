package partner

func isBusinessType(businessType string) bool {
	switch businessType {
	case "entertainment", "retail", "food_beverage", "spa", "excursion", "other":
		return true
	default:
		return false
	}
}

func isPartnerStatus(status string) bool {
	switch status {
	case "pending", "active", "suspended", "terminated":
		return true
	default:
		return false
	}
}

func isContractStatus(status string) bool {
	switch status {
	case "draft", "active", "expired", "terminated":
		return true
	default:
		return false
	}
}

func isPartnerStaffStatus(status string) bool {
	switch status {
	case "active", "inactive":
		return true
	default:
		return false
	}
}

func isSlotStatus(status string) bool {
	switch status {
	case "requested", "approved", "rejected", "cancelled":
		return true
	default:
		return false
	}
}
