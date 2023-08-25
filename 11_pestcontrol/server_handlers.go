package pestcontrol

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
)

var AuthorityAddr = "pestcontrol.protohackers.com:20547"

func HandleHello(conn io.Reader) error {
	msgType, contents, err := ReadMessage(conn)
	if err != nil {
		return err
	} else if msgType != 0x50 {
		return fmt.Errorf("expected Hello, got %x", msgType)
	}

	r := bytes.NewBuffer(contents)

	// protocol: str
	protocol, err := readString(r)
	if err != nil {
		return fmt.Errorf("reading protocol: %w", err)
	}

	if protocol != "pestcontrol" {
		return fmt.Errorf("invalid protocol: %q", protocol)
	}

	// version: u32
	version, err := readU32(r)
	if err != nil {
		return fmt.Errorf("reading version: %w", err)
	}

	if version != 1 {
		return fmt.Errorf("invalid version: %d", version)
	}

	if r.Len() != 0 {
		return fmt.Errorf("message has %d unused bytes", r.Len())
	}

	log.Printf("<-- Hello\n")

	return nil
}

func HandleSiteVisit(conn net.Conn) error {
	msgType, contents, err := ReadMessage(conn)
	if err != nil {
		return err
	} else if msgType != 0x58 {
		return fmt.Errorf("expected SiteVisit, got %x", msgType)
	}

	r := bytes.NewBuffer(contents)

	// site: u32
	site, err := readU32(r)
	if err != nil {
		return fmt.Errorf("reading site: %w", err)
	}

	// populations: [Observation]
	populations, err := readObservationArray(r)
	if err != nil {
		return fmt.Errorf("reading populations: %w", err)
	}

	if r.Len() != 0 {
		return fmt.Errorf("message has %d unused bytes", r.Len())
	}

	log.Printf("<-- SiteVisit{site:%d, populations:%#v}\n", site, populations)

	// Push observation to channel per site
	return ObserveSite(site, populations)
}

func HandleTargetPopulations(conn net.Conn) (uint32, []Target, error) {
	msgType, contents, err := ReadMessage(conn)
	if err != nil {
		return 0, nil, err
	}
	if msgType != 0x54 {
		return 0, nil, fmt.Errorf("expected TargetPopulations, got %x", msgType)
	}

	r := bytes.NewBuffer(contents)

	// site: u32
	site, err := readU32(r)
	if err != nil {
		return 0, nil, fmt.Errorf("reading site: %w", err)
	}

	// populations: [Target]
	populations, err := readTargetArray(r)
	if err != nil {
		return 0, nil, fmt.Errorf("reading populations: %w", err)
	}

	if r.Len() != 0 {
		return 0, nil, fmt.Errorf("message has %d unused bytes", r.Len())
	}

	log.Printf("<-- TargetPopulations{site:%d, populations:%v}\n", site, populations)

	return site, populations, nil
}

func HandlePolicyResult(conn net.Conn) (uint32, error) {
	msgType, contents, err := ReadMessage(conn)
	if err != nil {
		return 0, err
	}
	if msgType != 0x57 {
		return 0, fmt.Errorf("expected PolicyResult, got %x", msgType)
	}

	r := bytes.NewBuffer(contents)

	// policyID: u32
	policyID, err := readU32(r)
	if err != nil {
		return 0, fmt.Errorf("reading policyID: %w", err)
	}

	if r.Len() != 0 {
		return 0, fmt.Errorf("message has %d unused bytes", r.Len())
	}

	log.Printf("<-- PolicyResult{policyID:%d}\n", policyID)

	return policyID, nil
}

func HandleOK(conn net.Conn) error {
	msgType, contents, err := ReadMessage(conn)
	if err != nil {
		return err
	}
	if msgType != 0x52 {
		return fmt.Errorf("expected OK, got %x", msgType)
	}
	if len(contents) != 0 {
		return fmt.Errorf("message has %d unused bytes", len(contents))
	}

	log.Printf("<-- OK\n")

	return nil
}
