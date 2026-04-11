package robots

import "github.com/jimsmart/grobotstxt"

func (r Robots) IsAllow(agent string, uri string) bool {
	return grobotstxt.AgentAllowed(*r.Raw, agent, uri)
}
