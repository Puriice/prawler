package robots

import "github.com/purrice/prawler/internal/repository"

type RobotParser struct {
	repo repository.RobotsRepository
}

func NewRobotParser(repo repository.RobotsRepository) RobotParser {
	return RobotParser{repo: repo}
}
