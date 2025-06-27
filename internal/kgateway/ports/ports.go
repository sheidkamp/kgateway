package ports

//const portOffset = 8000

// Keep this function for now, as we may need to use the logic during upgrades
func TranslatePort(u uint16) uint16 {
	// if u >= 1024 {
	// 	return u
	// }
	// return u + portOffset
	return u
}
