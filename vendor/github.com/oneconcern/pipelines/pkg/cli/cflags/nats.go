package cflags

import (
	"os"
	"strings"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/oneconcern/pipelines/pkg/cli/envk"
	"github.com/spf13/cobra"
)

// ToCobraCommander interface for structs that can register themselves with cobra commands
type ToCobraCommander interface {
	RegisterFlags(cmd *cobra.Command)
}

// NATS for the flags with the common nats configuration
type NATS struct {
	ClusterID  string
	ClientID   string
	NatsURL    []string
	ClientCert string
	ClientKey  string
	RootCAs    []string
}

// RegisterFlags registers the cobra flags to populate this struct
func (n *NATS) RegisterFlags(cmd *cobra.Command) {
	fls := cmd.Flags()
	fls.StringVar(&n.ClusterID, "cluster", envk.StringOrDefault("NATS_CLUSTER", "test-cluster"), "The NATS Streaming cluster ID")
	fls.StringVar(&n.ClientID, "id", "", "The NATS Streaming client ID to connect with")
	cmd.MarkFlagRequired("id")
	fls.StringSliceVar(
		&n.NatsURL,
		"nats",
		envk.StringSliceOrDefault("NATS_SERVERS", "", []string{stan.DefaultNatsURL}),
		"the nats server urls",
	)
	fls.StringVar(&n.ClientCert, "nats-client-cert", "", "The client certificate to authenticate with nats with")
	cmd.MarkFlagFilename("nats-client-cert")
	fls.StringVar(&n.ClientKey, "nats-client-key", "", "The client private key to authenticate with nats with")
	cmd.MarkFlagFilename("nats-client-key")
	fls.StringSliceVar(&n.RootCAs, "nats-ca", nil, "the CA to use when verifying nats tls connections")
}

func (n *NATS) CreateConn() (stan.Conn, error) {
	servers := n.NatsURL
	if len(servers) == 0 {
		servers = []string{stan.DefaultNatsURL}
	}
	natsURI := strings.Join(servers, ",")
	opts := []nats.Option{nats.Name(n.ClientID)}
	if len(n.RootCAs) > 0 {
		opts = append(opts, nats.RootCAs(n.RootCAs...))
	}
	if n.ClientCert != "" && n.ClientKey != "" {
		opts = append(opts, nats.ClientCert(n.ClientCert, n.ClientKey))
	}
	nc, err := nats.Connect(natsURI, opts...)
	if err != nil {
		return nil, err
	}

	return stan.Connect(n.ClusterID, n.ClientID, stan.NatsConn(nc))
}

// AddTopicFlag adds the topic flag for the specified topic variable pointer
func AddTopicFlag(topic *string, cmd *cobra.Command) {
	fls := cmd.Flags()
	fls.StringVar(
		topic,
		"topic",
		os.Getenv("NATS_TOPIC"),
		"the topic to attach to. a publisher publishes into this, a listener receives the published events from this.")
	cmd.MarkFlagRequired("topic")
}

// AddGroupFlag adds the group flag for the specified group variable pointer
func AddGroupFlag(group *string, cmd *cobra.Command) {
	fls := cmd.Flags()
	fls.StringVar(
		group,
		"group",
		os.Getenv("NATS_GROUP"),
		"the group this listener belongs to")
}
