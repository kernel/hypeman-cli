package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var ingressCmd = cli.Command{
	Name:    "ingress",
	Aliases: []string{"ingresses"},
	Usage:   "Manage ingresses",
	Commands: []*cli.Command{
		&ingressCreateCmd,
		&ingressListCmd,
		&ingressGetCmd,
		&ingressDeleteCmd,
	},
	HideHelpCommand: true,
}

var ingressCreateCmd = cli.Command{
	Name:      "create",
	Usage:     "Create an ingress for an instance",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "hostname",
			Aliases: []string{"H"},
			Usage:   "Hostname to match (exact match on Host header)",
		},
		&cli.IntFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "Target port on the instance",
		},
		&cli.IntFlag{
			Name:  "host-port",
			Usage: "Host port to listen on (default: 80)",
			Value: 80,
		},
		&cli.BoolFlag{
			Name:  "tls",
			Usage: "Enable TLS termination (certificate auto-issued via ACME)",
		},
		&cli.BoolFlag{
			Name:  "redirect-http",
			Usage: "Auto-create HTTP to HTTPS redirect (only applies when --tls is enabled)",
		},
		&cli.StringFlag{
			Name:  "request-header-auth-header",
			Usage: "Request header that must match before proxying (reserved authentication, cookie, host, framing, proxy, and hop-by-hop headers are not allowed)",
		},
		&cli.StringFlag{
			Name:  "request-header-auth-value",
			Usage: "Exact header value required before proxying",
		},
		&cli.StringSliceFlag{
			Name: "rule",
			Usage: "Add a routing rule (can be repeated): " + ingressRuleSpecFormat + ". " +
				"Omit the instance to target the positional <instance>. When any --rule is given, the single-rule " +
				"shorthand flags (--hostname/--port/--host-port/--tls/--redirect-http/--request-header-auth-header/" +
				"--request-header-auth-value) must not be used.",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Ingress name (auto-generated from hostname if not provided)",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Set ingress tag key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleIngressCreate,
	HideHelpCommand: true,
}

var ingressListCmd = cli.Command{
	Name:  "list",
	Usage: "List ingresses",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display ingress IDs",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Filter by tag key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleIngressList,
	HideHelpCommand: true,
}

var ingressGetCmd = cli.Command{
	Name:            "get",
	Usage:           "Get ingress details",
	ArgsUsage:       "<id>",
	Action:          handleIngressGet,
	HideHelpCommand: true,
}

var ingressDeleteCmd = cli.Command{
	Name:            "delete",
	Usage:           "Delete an ingress",
	ArgsUsage:       "<id>",
	Action:          handleIngressDelete,
	HideHelpCommand: true,
}

func handleIngressCreate(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance name or ID required\nUsage: hypeman ingress create <instance> --hostname <hostname> --port <port>")
	}

	instance := args[0]
	name := cmd.String("name")

	ruleSpecs := cmd.StringSlice("rule")
	var rules []hypeman.IngressRuleParam
	var primaryHostname string
	if len(ruleSpecs) > 0 {
		for _, flag := range []string{"hostname", "port", "host-port", "tls", "redirect-http", "request-header-auth-header", "request-header-auth-value"} {
			if cmd.IsSet(flag) {
				return fmt.Errorf("--rule cannot be combined with --%s; provide all rules via --rule", flag)
			}
		}
		for _, spec := range ruleSpecs {
			rule, err := parseIngressRuleSpec(spec, instance)
			if err != nil {
				return fmt.Errorf("invalid rule %q: %w", spec, err)
			}
			rules = append(rules, rule)
		}
		primaryHostname = rules[0].Match.Hostname
	} else {
		hostname := cmd.String("hostname")
		if hostname == "" {
			return fmt.Errorf("--hostname is required (or use --rule)")
		}
		if !cmd.IsSet("port") {
			return fmt.Errorf("--port is required (or use --rule)")
		}
		rule := hypeman.IngressRuleParam{
			Match: hypeman.IngressMatchParam{
				Hostname: hostname,
				Port:     hypeman.Int(int64(cmd.Int("host-port"))),
			},
			Target: hypeman.IngressTargetParam{
				Instance: instance,
				Port:     int64(cmd.Int("port")),
			},
			Tls:          hypeman.Bool(cmd.Bool("tls")),
			RedirectHTTP: hypeman.Bool(cmd.Bool("redirect-http")),
		}
		auth, err := requestHeaderAuthFromFlags(cmd.String("request-header-auth-header"), cmd.String("request-header-auth-value"))
		if err != nil {
			return err
		}
		if auth != nil {
			rule.RequestHeaderAuth = *auth
		}
		rules = []hypeman.IngressRuleParam{rule}
		primaryHostname = hostname
	}

	// Auto-generate name from hostname if not provided
	if name == "" {
		name = generateIngressName(primaryHostname)
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	params := hypeman.IngressNewParams{
		Name:  name,
		Rules: rules,
	}
	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag: %s\n", malformed)
	}
	if len(tags) > 0 {
		params.Tags = tags
	}

	fmt.Fprintf(os.Stderr, "Creating ingress %s...\n", name)

	result, err := client.Ingresses.New(ctx, params, opts...)
	if err != nil {
		return err
	}

	fmt.Println(result.ID)
	return nil
}

func handleIngressList(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	params := hypeman.IngressListParams{}
	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag filter: %s\n", malformed)
	}
	if len(tags) > 0 {
		params.Tags = tags
	}

	// If a specific format is requested (not "auto"), output in that format
	if format != "auto" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		_, err := client.Ingresses.List(ctx, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "ingress list", obj, format, transform)
	}

	ingresses, err := client.Ingresses.List(ctx, params, opts...)
	if err != nil {
		return err
	}

	quietMode := cmd.Bool("quiet")

	if quietMode {
		for _, ing := range *ingresses {
			fmt.Println(ing.ID)
		}
		return nil
	}

	if len(*ingresses) == 0 {
		fmt.Fprintln(os.Stderr, "No ingresses found.")
		return nil
	}

	table := NewTableWriter(os.Stdout, "ID", "NAME", "HOSTNAME", "TARGET", "TLS", "CREATED")
	table.TruncOrder = []int{2, 3, 5, 1} // HOSTNAME first, then TARGET, CREATED, NAME
	for _, ing := range *ingresses {
		// Extract first rule's hostname and target for display
		hostname := ""
		target := ""
		tlsEnabled := "-"
		if len(ing.Rules) > 0 {
			rule := ing.Rules[0]
			hostname = rule.Match.Hostname
			target = fmt.Sprintf("%s:%d", rule.Target.Instance, rule.Target.Port)
			if rule.Tls {
				tlsEnabled = "yes"
			} else {
				tlsEnabled = "no"
			}
		}

		table.AddRow(
			TruncateID(ing.ID),
			ing.Name,
			hostname,
			target,
			tlsEnabled,
			FormatTimeAgo(ing.CreatedAt),
		)
	}
	table.Render()

	return nil
}

func handleIngressGet(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("ingress ID required\nUsage: hypeman ingress get <id>")
	}

	id := args[0]

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err := client.Ingresses.Get(ctx, id, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)
	return ShowJSON(os.Stdout, "ingress get", obj, format, transform)
}

func handleIngressDelete(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("ingress ID or name required\nUsage: hypeman ingress delete <id>")
	}

	id := args[0]

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	err := client.Ingresses.Delete(ctx, id, opts...)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Deleted ingress %s\n", id)
	return nil
}

// ingressRuleSpecFormat documents the --rule grammar shared by the flag usage
// text and the parser error message.
const ingressRuleSpecFormat = "hostname[:host-port]=instance:port[,tls][,redirect-http][,request-header-auth=HEADER:VALUE]"

// parseIngressRuleSpec parses a routing rule specification string in the
// ingressRuleSpecFormat grammar. When the instance is omitted (e.g.
// "host:80=:8080"), fallbackInstance is used.
func parseIngressRuleSpec(spec, fallbackInstance string) (hypeman.IngressRuleParam, error) {
	matchPart, targetPart, ok := strings.Cut(spec, "=")
	if !ok {
		return hypeman.IngressRuleParam{}, fmt.Errorf("expected format %s", ingressRuleSpecFormat)
	}

	hostname, hostPortStr, hasHostPort := strings.Cut(matchPart, ":")
	if hostname == "" {
		return hypeman.IngressRuleParam{}, fmt.Errorf("hostname cannot be empty")
	}
	hostPort := int64(80)
	if hasHostPort {
		parsed, err := strconv.ParseInt(hostPortStr, 10, 64)
		if err != nil {
			return hypeman.IngressRuleParam{}, fmt.Errorf("invalid host port %q: %w", hostPortStr, err)
		}
		hostPort = parsed
	}

	targetSegments := strings.Split(targetPart, ",")
	targetSpec := targetSegments[0]
	targetInstance, portStr, ok := strings.Cut(targetSpec, ":")
	if !ok {
		return hypeman.IngressRuleParam{}, fmt.Errorf("target must be instance:port")
	}
	if targetInstance == "" {
		targetInstance = fallbackInstance
	}
	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		return hypeman.IngressRuleParam{}, fmt.Errorf("invalid target port %q: %w", portStr, err)
	}

	rule := hypeman.IngressRuleParam{
		Match: hypeman.IngressMatchParam{
			Hostname: hostname,
			Port:     hypeman.Int(hostPort),
		},
		Target: hypeman.IngressTargetParam{
			Instance: targetInstance,
			Port:     port,
		},
	}

	for _, opt := range targetSegments[1:] {
		switch {
		case opt == "":
			continue
		case opt == "tls":
			rule.Tls = hypeman.Bool(true)
		case opt == "redirect-http":
			rule.RedirectHTTP = hypeman.Bool(true)
		case strings.HasPrefix(opt, requestHeaderAuthOption+"="):
			header, value, _ := strings.Cut(strings.TrimPrefix(opt, requestHeaderAuthOption+"="), ":")
			auth, err := requestHeaderAuthFromFlags(header, value)
			if err != nil {
				return hypeman.IngressRuleParam{}, err
			}
			if auth == nil {
				return hypeman.IngressRuleParam{}, fmt.Errorf("%s must be HEADER:VALUE", requestHeaderAuthOption)
			}
			rule.RequestHeaderAuth = *auth
		default:
			return hypeman.IngressRuleParam{}, fmt.Errorf("unknown option %q", opt)
		}
	}

	return rule, nil
}

const requestHeaderAuthOption = "request-header-auth"

// requestHeaderAuthFromFlags builds the request header auth param, returning nil
// when neither half was supplied. Both halves are required by the API.
func requestHeaderAuthFromFlags(header, value string) (*hypeman.IngressRuleRequestHeaderAuthParam, error) {
	if header == "" && value == "" {
		return nil, nil
	}
	if header == "" {
		return nil, fmt.Errorf("request header auth requires a header name")
	}
	if value == "" {
		return nil, fmt.Errorf("request header auth requires a value for header %q", header)
	}
	return &hypeman.IngressRuleRequestHeaderAuthParam{Header: header, Value: value}, nil
}

// generateIngressName generates an ingress name from hostname
func generateIngressName(hostname string) string {
	// Replace dots with dashes
	name := strings.ReplaceAll(hostname, ".", "-")
	name = strings.ToLower(name)

	// Remove invalid characters (only allow a-z, 0-9, and -)
	var cleaned strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned.WriteRune(r)
		}
	}
	name = cleaned.String()

	// Trim leading/trailing dashes
	name = strings.Trim(name, "-")

	// Add random suffix
	suffix := randomSuffix(4)
	return fmt.Sprintf("%s-%s", name, suffix)
}
