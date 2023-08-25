package pestcontrol

import (
	"io"
	"log"
)

func WriteHello(w io.Writer) error {
	data := []byte{}
	data = append(data, stringToBytes("pestcontrol")...)
	data = append(data, uint32ToBytes(1)...)

	if err := writeMessage(w, 0x50, data); err != nil {
		return err
	}

	log.Printf("--> Hello\n")

	return nil
}

func WriteError(w io.Writer, err error) error {
	data := []byte{}
	data = append(data, stringToBytes(err.Error())...)

	if err := writeMessage(w, 0x51, data); err != nil {
		return err
	}

	log.Printf("--> Error{message:%q}\n", err.Error())

	return nil
}

func WriteDialAuthority(w io.Writer, site uint32) error {
	data := []byte{}
	data = append(data, uint32ToBytes(site)...)

	if err := writeMessage(w, 0x53, data); err != nil {
		return err
	}

	log.Printf("--> DialAuthority{site:%d}\n", site)

	return nil
}

func WriteCreatePolicy(w io.Writer, species string, action byte) error {
	data := []byte{}
	data = append(data, stringToBytes(species)...)
	data = append(data, action)

	if err := writeMessage(w, 0x55, data); err != nil {
		return err
	}

	actionStr := "inaction"
	if action == CullPolicy {
		actionStr = "cull"
	} else if action == ConservePolicy {
		actionStr = "conserve"
	} else {
		actionStr = "bad"
	}
	log.Printf("--> CreatePolicy{species:%q, action:%s}\n", species, actionStr)

	return nil
}

const CullPolicy = 0x90
const ConservePolicy = 0xa0

func WriteDeletePolicy(w io.Writer, policyID uint32) error {
	data := []byte{}
	data = append(data, uint32ToBytes(policyID)...)

	if err := writeMessage(w, 0x56, data); err != nil {
		return err
	}

	log.Printf("--> DeletePolicy{policyID:%d}\n", policyID)

	return nil
}

// ===== tests =====
func WriteSiteVisit(w io.Writer, site uint32, observations []Observation) error {
	data := []byte{}
	data = append(data, uint32ToBytes(site)...)
	data = append(data, uint32ToBytes(uint32(len(observations)))...)
	for _, observation := range observations {
		data = append(data, stringToBytes(observation.Species)...)
		data = append(data, uint32ToBytes(observation.Count)...)
	}

	if err := writeMessage(w, 0x58, data); err != nil {
		return err
	}

	log.Printf("--> SiteVisit{site:%d, observations:%v}\n", site, observations)

	return nil
}
