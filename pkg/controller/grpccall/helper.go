package grpccall

import (
	"fmt"
	"github.com/go-logr/logr"
	upmv1alpha1 "github.com/upmio/unit-operator/api/v1alpha1"
	upmv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/agent/app/mysql"
	"github.com/upmio/unit-operator/pkg/agent/app/postgresql"
	"github.com/upmio/unit-operator/pkg/agent/app/proxysql"
	"github.com/upmio/unit-operator/pkg/agent/app/redis"
	"github.com/upmio/unit-operator/pkg/agent/app/sentinel"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"

	"context"
	"google.golang.org/grpc"
	"net"
)

type Client struct {
	conn *grpc.ClientConn
}

// newGrpcClient builds grpc client to call unit agent interface
func newGrpcClient(host, port string) (*Client, error) {
	//conn, err := grpc.Dial(conf.Address(), grpc.WithInsecure(), grpc.WithPerRPCCredentials(conf.Authentication))
	conn, err := grpc.NewClient(net.JoinHostPort(host, port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn: conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Mysql sdk
func (c *Client) Mysql() mysql.MysqlOperationClient {
	return mysql.NewMysqlOperationClient(c.conn)
}

// Postgresql sdk
func (c *Client) Postgresql() postgresql.PostgresqlOperationClient {
	return postgresql.NewPostgresqlOperationClient(c.conn)
}

// Proxysql sdk
func (c *Client) Proxysql() proxysql.ProxysqlOperationClient {
	return proxysql.NewProxysqlOperationClient(c.conn)
}

// Redis sdk
func (c *Client) Redis() redis.RedisOperationClient {
	return redis.NewRedisOperationClient(c.conn)
}

// Sentinel sdk
func (c *Client) Sentinel() sentinel.SentinelOperationClient {
	return sentinel.NewSentinelOperationClient(c.conn)
}

// gatherUnitAgentEndpoint retrieves and returns the host and port for the unit-agent container.
func gatherUnitAgentEndpoint(
	ctx context.Context,
	client client.Client,
	instance *upmv1alpha1.GrpcCall,
	reqLogger logr.Logger,
) (string, string, error) {
	// 1. Retrieve the Unit object
	unit := &upmv1alpha2.Unit{}
	key := types.NamespacedName{
		Name:      instance.Spec.TargetUnit,
		Namespace: instance.Namespace,
	}
	if err := client.Get(ctx, key, unit); err != nil {
		return "", "", fmt.Errorf("failed to fetch unit [%s]: %v", key, err)
	}

	// 2. Construct the host DNS name
	host := fmt.Sprintf("%s.%s.%s.svc", unit.Name, upmv1alpha2.UnitsetHeadlessSvcName(unit), unit.Namespace)

	// 3. Find the container named "unit-agent"
	var agent *corev1.Container
	for i := range unit.Spec.Template.Spec.Containers {
		c := &unit.Spec.Template.Spec.Containers[i]
		if c.Name == agentName {
			agent = c
			break
		}
	}
	if agent == nil {
		return "", "", fmt.Errorf("container %q not found in unit %q", "unit-agent", key)
	}

	// 4. Locate the port named "unit-agent" within the container
	var agentPort *corev1.ContainerPort
	for i := range agent.Ports {
		p := &agent.Ports[i]
		if p.Name == agentName {
			agentPort = p
			break
		}
	}
	if agentPort == nil {
		return "", "", fmt.Errorf("port %s not found in container", agentName)
	}

	// 5. Return the host and port as strings
	port := strconv.Itoa(int(agentPort.ContainerPort))
	return host, port, nil
}
