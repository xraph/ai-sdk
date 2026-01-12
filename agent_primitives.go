package sdk

import (
	"context"
	"errors"
	"fmt"
	"sync"

	logger "github.com/xraph/go-utils/log"
)

// AgentOrchestrator coordinates multiple agents.
type AgentOrchestrator struct {
	agents map[string]*Agent
	router AgentOrchestratorRouter
	logger logger.Logger
	mu     sync.RWMutex
}

// AgentOrchestratorRouter routes requests to agents within an orchestrator.
type AgentOrchestratorRouter interface {
	Route(ctx context.Context, input string, agents map[string]*Agent) (*Agent, error)
}

// NewAgentOrchestrator creates a new orchestrator.
func NewAgentOrchestrator(logger logger.Logger) *AgentOrchestrator {
	return &AgentOrchestrator{
		agents: make(map[string]*Agent),
		logger: logger,
	}
}

// AddAgent adds an agent to the orchestrator.
func (o *AgentOrchestrator) AddAgent(name string, agent *Agent) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.agents[name] = agent
}

// RemoveAgent removes an agent.
func (o *AgentOrchestrator) RemoveAgent(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.agents, name)
}

// GetAgent returns an agent by name.
func (o *AgentOrchestrator) GetAgent(name string) (*Agent, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agent, ok := o.agents[name]

	return agent, ok
}

// SetRouter sets the router.
func (o *AgentOrchestrator) SetRouter(router AgentOrchestratorRouter) {
	o.router = router
}

// Execute routes and executes with an appropriate agent.
func (o *AgentOrchestrator) Execute(ctx context.Context, input string) (*AgentExecution, error) {
	o.mu.RLock()

	agents := make(map[string]*Agent)
	for k, v := range o.agents {
		agents[k] = v
	}

	o.mu.RUnlock()

	if len(agents) == 0 {
		return nil, errors.New("no agents available")
	}

	var agent *Agent

	if o.router != nil {
		var err error

		agent, err = o.router.Route(ctx, input, agents)
		if err != nil {
			return nil, fmt.Errorf("routing failed: %w", err)
		}
	} else {
		// Use first agent if no router
		for _, a := range agents {
			agent = a

			break
		}
	}

	return agent.ExecuteWithSteps(ctx, input)
}

// ExecuteSequential executes multiple agents in sequence.
func (o *AgentOrchestrator) ExecuteSequential(ctx context.Context, input string, agentNames ...string) ([]*AgentExecution, error) {
	executions := make([]*AgentExecution, 0, len(agentNames))
	currentInput := input

	for _, name := range agentNames {
		agent, ok := o.GetAgent(name)
		if !ok {
			return executions, fmt.Errorf("agent not found: %s", name)
		}

		execution, err := agent.ExecuteWithSteps(ctx, currentInput)
		executions = append(executions, execution)

		if err != nil {
			return executions, err
		}

		currentInput = execution.FinalOutput
	}

	return executions, nil
}

// ExecuteParallel executes multiple agents in parallel.
func (o *AgentOrchestrator) ExecuteParallel(ctx context.Context, input string, agentNames ...string) ([]*AgentExecution, error) {
	var wg sync.WaitGroup

	executions := make([]*AgentExecution, len(agentNames))
	errors := make([]error, len(agentNames))

	for i, name := range agentNames {
		wg.Add(1)

		go func(idx int, agentName string) {
			defer wg.Done()

			agent, ok := o.GetAgent(agentName)
			if !ok {
				errors[idx] = fmt.Errorf("agent not found: %s", agentName)

				return
			}

			execution, err := agent.ExecuteWithSteps(ctx, input)
			executions[idx] = execution
			errors[idx] = err
		}(i, name)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return executions, err
		}
	}

	return executions, nil
}
