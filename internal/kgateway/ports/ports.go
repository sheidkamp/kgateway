package ports

import "fmt"

const PortOffset = 8000

type Ports struct {
	translatePort func(u uint16) uint16
}

var ports *Ports

func TranslatePort(u uint16) uint16 {
	if ports == nil {
		panic("port translator not initialized")
	}
	return ports.TranslatePort(u)
}

// Keep this function for now, as we may need to use the logic during upgrades
func translatePort(u uint16) uint16 {
	if u >= 1024 {
		return u
	}
	return u + PortOffset
}

func noPortTranslation(u uint16) uint16 {
	return u
}

func (p *Ports) TranslatePort(u uint16) uint16 {
	return p.translatePort(u)
}

func Init(disablePortMapping bool) {
	fmt.Printf("Initializing port translator with disablePortMapping: %v\n", disablePortMapping)
	if disablePortMapping {
		ports = &Ports{
			translatePort: noPortTranslation,
		}
	} else {
		ports = &Ports{
			translatePort: translatePort,
		}
	}
}
