package dockerfile

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Protocol is the transport protocol associated with an exposed port.
type Protocol uint8

const (
	ProtoTCP Protocol = iota
	ProtoUDP
)

type ExposedPort interface{ isExposedPort() }

type Port struct {
	Number uint16
	Proto  Protocol
}
type PortRange struct {
	From  uint16
	To    uint16
	Proto Protocol
}

func (Port) isExposedPort()      {}
func (PortRange) isExposedPort() {}

func TCP(number uint16) Port { return Port{Number: number, Proto: ProtoTCP} }
func UDP(number uint16) Port { return Port{Number: number, Proto: ProtoUDP} }
func TCPRange(from, to uint16) PortRange {
	return PortRange{From: from, To: to, Proto: ProtoTCP}
}
func UDPRange(from, to uint16) PortRange {
	return PortRange{From: from, To: to, Proto: ProtoUDP}
}

// Expose adds port metadata to the image config. It has no filesystem effect.
func Expose(ports ...ExposedPort) Instruction {
	values := append([]ExposedPort(nil), ports...)
	return NewInstruction("EXPOSE", func(_ context.Context, state *State, _ *Runtime, _ InstructionMetadata) error {
		if len(values) == 0 {
			return errors.New("EXPOSE requires at least one port")
		}
		if state.Image.Config.ExposedPorts == nil {
			state.Image.Config.ExposedPorts = make(map[string]struct{})
		}
		for _, value := range values {
			switch port := value.(type) {
			case Port:
				if port.Number == 0 {
					return errors.New("EXPOSE port must be between 1 and 65535")
				}
				state.Image.Config.ExposedPorts[portKey(port.Number, port.Proto)] = struct{}{}
			case PortRange:
				if port.From == 0 || port.To < port.From {
					return fmt.Errorf("invalid EXPOSE port range %d-%d", port.From, port.To)
				}
				for current := uint32(port.From); current <= uint32(port.To); current++ {
					state.Image.Config.ExposedPorts[portKey(uint16(current), port.Proto)] = struct{}{}
				}
			default:
				return fmt.Errorf("unsupported exposed port type %T", value)
			}
		}
		return nil
	}, WithDescription("EXPOSE "+fmt.Sprint(values)),
		WithOnBuildExpression(func() (string, error) {
			if len(values) == 0 {
				return "", errors.New("EXPOSE requires ports")
			}
			var parts []string
			for _, value := range values {
				switch port := value.(type) {
				case Port:
					if port.Number == 0 {
						return "", errors.New("EXPOSE port cannot be zero")
					}
					parts = append(parts, portKey(port.Number, port.Proto))
				case PortRange:
					if port.From == 0 || port.To < port.From {
						return "", errors.New("invalid EXPOSE port range")
					}
					for current := uint32(port.From); current <= uint32(port.To); current++ {
						parts = append(parts, portKey(uint16(current), port.Proto))
					}
				default:
					return "", fmt.Errorf("unsupported exposed port type %T", value)
				}
			}
			return "EXPOSE " + strings.Join(parts, " "), nil
		}))
}

func portKey(number uint16, protocol Protocol) string {
	suffix := "tcp"
	if protocol == ProtoUDP {
		suffix = "udp"
	}
	return strconv.Itoa(int(number)) + "/" + suffix
}

