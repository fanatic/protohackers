package pestcontrol

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type Policy struct {
	Action byte
	ID     uint32
}

var SiteObservations map[uint32]chan []Observation = make(map[uint32]chan []Observation)
var SiteObservationsMutex sync.Mutex

func ObserveSite(site uint32, observation []Observation) error {
	// Check for conflicting observations
	species := map[string]uint32{}
	for _, o := range observation {
		if count := species[o.Species]; count > 0 && count != o.Count {
			return fmt.Errorf("conflicting observations for %q: %d != %d", o.Species, count, o.Count)
		}
		species[o.Species] = o.Count
	}

	SiteObservationsMutex.Lock()
	defer SiteObservationsMutex.Unlock()

	if _, ok := SiteObservations[site]; !ok {
		SiteObservations[site] = make(chan []Observation)
		go func() {
			if err := WatchSite(site, SiteObservations[site]); err != nil {
				log.Printf("policies at=watch_site.error site=%d error=%q\n", site, err.Error())
			}
		}()
	}
	SiteObservations[site] <- observation
	return nil
}

func WatchSite(site uint32, ch chan []Observation) error {
	log.Printf("policies at=watch_site.start site=%d\n", site)
	// Connect to the authority for the specified site
	authorityServer, err := net.Dial("tcp", AuthorityAddr)
	if err != nil {
		return err
	}
	defer authorityServer.Close()

	// Send Hello
	if err := WriteHello(authorityServer); err != nil {
		return err
	}
	log.Printf("policies at=watch_site.sent-hello site=%d\n", site)

	// Expect Hello
	if err := HandleHello(authorityServer); err != nil {
		return err
	}
	log.Printf("policies at=watch_site.got-hello site=%d\n", site)

	// Send DialAuthority
	if err := WriteDialAuthority(authorityServer, site); err != nil {
		return err
	}
	log.Printf("policies at=watch_site.sent-dial site=%d\n", site)

	// Expect TargetPopulations
	targetSite, targets, err := HandleTargetPopulations(authorityServer)
	if err != nil {
		return err
	}
	if targetSite != site {
		return fmt.Errorf("expected site %d, got %d", site, targetSite)
	}
	log.Printf("policies at=watch_site.got-targets site=%d\n", site)

	policyCache := make(map[string]Policy)

	for populations := range ch {
		log.Printf("policies at=watch_site.applying-policy site=%d p=%d\n", site, len(populations))
		if err := applyPolicyRules(site, targets, populations, authorityServer, &policyCache); err != nil {
			log.Printf("policies at=apply-rules.error site=%d error=%q\n", site, err.Error())
		}
	}
	return nil
}

func applyPolicyRules(site uint32, targets []Target, populations []Observation, authorityServer net.Conn, policyCache *map[string]Policy) error {
	species := map[string]uint32{}
	// Reset species cache to latest observations
	for _, target := range targets {
		species[target.Species] = 0
	}
	for _, observation := range populations {
		species[observation.Species] = observation.Count
	}

	for _, target := range targets {
		for species, observation := range species {
			if species == target.Species {
				if err := applySpeciesPolicy(authorityServer, site, observation, target, policyCache); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func applySpeciesPolicy(authorityServer net.Conn, site uint32, observationCount uint32, target Target, policyCache *map[string]Policy) error {
	// Check for existing policy
	policy, ok := (*policyCache)[target.Species]

	// If there is no change from an existing action, do nothing
	if ok && policy.Action == ConservePolicy && observationCount < target.Min {
		return nil
	} else if ok && policy.Action == CullPolicy && observationCount > target.Max {
		return nil
	}

	newAction := "inaction"

	// Delete existing policy before creating a new one
	if ok && (policy.Action == ConservePolicy && observationCount >= target.Min ||
		policy.Action == CullPolicy && observationCount <= target.Max) {
		if err := WriteDeletePolicy(authorityServer, policy.ID); err != nil {
			return err
		}

		// Expect OK
		if err := HandleOK(authorityServer); err != nil {
			return err
		}

		delete(*policyCache, target.Species)
	}

	if observationCount < target.Min || observationCount > target.Max {
		action := byte(CullPolicy)
		newAction = "cull"
		if observationCount < target.Min {
			action = ConservePolicy
			newAction = "conserve"
		}

		if err := WriteCreatePolicy(authorityServer, target.Species, action); err != nil {
			return err
		}

		// Expect PolicyResult
		policyID, err := HandlePolicyResult(authorityServer)
		if err != nil {
			return err
		}

		// Remember policy
		(*policyCache)[target.Species] = Policy{action, policyID}
	}

	cachedAction := "inaction"
	if ok && policy.Action == ConservePolicy {
		cachedAction = "conserve"
	} else if ok && policy.Action == CullPolicy {
		cachedAction = "cull"
	}

	log.Printf("--- %d %s cached-action:%s new-action:%s count:%d target:%d-%d\n", site, target.Species, cachedAction, newAction, observationCount, target.Min, target.Max)

	return nil
}
