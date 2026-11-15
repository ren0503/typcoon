package booking

func isServiceBookingStatus(status string) bool {
	switch status {
	case "pending", "confirmed", "rejected", "cancelled", "completed":
		return true
	default:
		return false
	}
}
