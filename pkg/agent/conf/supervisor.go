package conf

import (
	"fmt"
	supervisord "github.com/abrander/go-supervisord"
)

var supervisorClient *supervisord.Client

func (s *Supervisor) GetSupervisorClient() (*supervisord.Client, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if supervisorClient == nil {
		conn, err := s.getSupervisorClient()
		if err != nil {
			return nil, err
		}
		supervisorClient = conn
	}
	return supervisorClient, nil
}

func (s *Supervisor) getSupervisorClient() (*supervisord.Client, error) {
	return supervisord.NewClient(fmt.Sprintf("http://%s:%d/RPC2", s.Addr, s.Port))
}
