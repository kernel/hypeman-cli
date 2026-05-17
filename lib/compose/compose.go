package compose

import (
	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
)

const (
	composeTagName     = "hypeman.compose.name"
	composeTagService  = "hypeman.compose.service"
	composeTagResource = "hypeman.compose.resource"
	composeTagHash     = "hypeman.compose.hash"

	composeResourceInstance = "instance"
	composeResourceIngress  = "ingress"
)

type Runner struct {
	file   string
	spec   composeSpec
	client hypeman.Client
	opts   []option.RequestOption
}

type UpOptions struct {
	Replace     bool
	Wait        bool
	WaitTimeout string
	Verbose     bool
}

type Plan struct {
	Name    string   `json:"name"`
	File    string   `json:"file"`
	Actions []Action `json:"actions"`
	Summary Summary  `json:"summary"`
}

type Summary struct {
	Create    int `json:"create"`
	Replace   int `json:"replace"`
	Delete    int `json:"delete"`
	Unchanged int `json:"unchanged"`
	Skip      int `json:"skip"`
	Conflict  int `json:"conflict"`
}

type Action struct {
	Action  string `json:"action"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Service string `json:"service,omitempty"`
	Reason  string `json:"reason"`

	instanceID    string
	ingressID     string
	instanceInput hypeman.InstanceNewParams
	ingressInput  hypeman.IngressNewParams
}

func NewRunner(file string, client hypeman.Client, opts ...option.RequestOption) (*Runner, error) {
	spec, err := loadComposeSpec(file)
	if err != nil {
		return nil, err
	}
	return &Runner{
		file:   file,
		spec:   spec,
		client: client,
		opts:   opts,
	}, nil
}
