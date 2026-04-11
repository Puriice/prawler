package robots

import "github.com/jimsmart/grobotstxt"

func (r Robots) IsAllow(agent string, uri string) bool {
	if r.Raw == nil || *r.Raw == "" {
		return true
	}

	return grobotstxt.AgentAllowed(*r.Raw, agent, uri)
}
