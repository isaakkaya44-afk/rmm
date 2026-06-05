package service

import (
	"fmt"
	"log"
	"os/exec"

	"golang.org/x/sys/windows/svc"
)

type AgentService struct {
	stopCh chan bool
}

func NewAgentService() *AgentService {
	return &AgentService{
		stopCh: make(chan bool),
	}
}

func (s *AgentService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	go s.run()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				s.stopCh <- true
				return false, 0
			}
		}
	}
}

func (s *AgentService) run() {
	log.Println("RMM Agent service started")
	<-s.stopCh
	log.Println("RMM Agent service stopped")
}

func InstallService(name, displayName, binaryPath string) error {
	return runSC("create", name, fmt.Sprintf("binPath=%s", binaryPath),
		fmt.Sprintf("DisplayName=%s", displayName), "start=auto")
}

func RemoveService(name string) error {
	return runSC("delete", name)
}

func StartService(name string) error {
	return runSC("start", name)
}

func StopService(name string) error {
	return runSC("stop", name)
}

func runSC(args ...string) error {
	cmd := exec.Command("sc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sc %s failed: %s: %w", args[0], string(output), err)
	}
	return nil
}
