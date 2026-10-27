// Package serial manages attached Feather transceivers.
package serial

import (
	"regexp"
	"strconv"
)

var (
	rssiRe    = regexp.MustCompile(`RSSI: (-?\d+)`)
	decodedRe = regexp.MustCompile(`Decoded Value: (.+)`)
	// The Skinny-LoRa firmware prints this once from setup(), e.g. "Freq set to: 915.00".
	freqRe = regexp.MustCompile(`Freq set to: (\d+(?:\.\d+)?)`)
)

// ParseFreq extracts the radio frequency (MHz) from the firmware's boot banner.
func ParseFreq(line string) (float64, bool) {
	m := freqRe.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	f, err := strconv.ParseFloat(m[1], 64)
	return f, err == nil && f > 0
}

// Parser pairs the "RSSI:" and "Decoded Value:" lines the Skinny-LoRa
// firmware prints separately into one packet.
type Parser struct {
	rssi    *int
	decoded *string
}

// Feed consumes one serial line and reports a completed packet, if any.
func (p *Parser) Feed(line string) (rssi int, decoded string, ok bool) {
	if m := rssiRe.FindStringSubmatch(line); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil {
			p.rssi = &v
		}
	}
	if m := decodedRe.FindStringSubmatch(line); m != nil {
		d := m[1]
		p.decoded = &d
	}
	if p.rssi != nil && p.decoded != nil {
		rssi, decoded, ok = *p.rssi, *p.decoded, true
		p.rssi, p.decoded = nil, nil
	}
	return
}
