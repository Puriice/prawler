package robots

import "github.com/jimsmart/grobotstxt"

func (r Robots) IsAllow(agent string, uri string) bool {
	if r.Raw == nil {
		return false
	}

	return grobotstxt.AgentAllowed(*r.Raw, agent, uri)
}
