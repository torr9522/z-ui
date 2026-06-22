package runtime

import (
	"context"
	"fmt"
	"time"

	proxymancommand "github.com/xtls/xray-core/app/proxyman/command"
	commonprotocol "github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	xtcore "github.com/xtls/xray-core/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	apiPort int
	timeout time.Duration
}

func NewClient(apiPort int) *Client {
	return &Client{
		apiPort: apiPort,
		timeout: 5 * time.Second,
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.withHandlerService(ctx, func(handler proxymancommand.HandlerServiceClient, callCtx context.Context) (interface{}, error) {
		return handler.ListInbounds(callCtx, &proxymancommand.ListInboundsRequest{
			IsOnlyTags: true,
		})
	})
	return err
}

func (c *Client) Apply(ctx context.Context, plan *Plan) error {
	for _, operation := range plan.Operations {
		if err := c.applyOperation(ctx, operation); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) applyOperation(ctx context.Context, operation Operation) error {
	_, err := c.withHandlerService(ctx, func(handler proxymancommand.HandlerServiceClient, callCtx context.Context) (interface{}, error) {
		switch operation.Type {
		case OperationAddInbound:
			return handler.AddInbound(callCtx, &proxymancommand.AddInboundRequest{Inbound: operation.Inbound})
		case OperationRemoveInbound:
			return handler.RemoveInbound(callCtx, &proxymancommand.RemoveInboundRequest{Tag: operation.Tag})
		case OperationReplaceInbound:
			if _, err := handler.RemoveInbound(callCtx, &proxymancommand.RemoveInboundRequest{Tag: operation.Tag}); err != nil {
				return nil, err
			}
			return handler.AddInbound(callCtx, &proxymancommand.AddInboundRequest{Inbound: operation.Inbound})
		case OperationAddClient:
			return handler.AlterInbound(callCtx, &proxymancommand.AlterInboundRequest{
				Tag: operation.Tag,
				Operation: serial.ToTypedMessage(&proxymancommand.AddUserOperation{
					User: operation.User,
				}),
			})
		case OperationRemoveClient:
			return handler.AlterInbound(callCtx, &proxymancommand.AlterInboundRequest{
				Tag: operation.Tag,
				Operation: serial.ToTypedMessage(&proxymancommand.RemoveUserOperation{
					Email: operation.Email,
				}),
			})
		default:
			return nil, fmt.Errorf("unsupported runtime operation %q", operation.Type)
		}
	})
	return err
}

func (c *Client) withHandlerService(ctx context.Context, fn func(proxymancommand.HandlerServiceClient, context.Context) (interface{}, error)) (interface{}, error) {
	conn, callCtx, cancel, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer conn.Close()
	return fn(proxymancommand.NewHandlerServiceClient(conn), callCtx)
}

func (c *Client) dial(ctx context.Context) (*grpc.ClientConn, context.Context, context.CancelFunc, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	conn, err := grpc.DialContext(
		callCtx,
		fmt.Sprintf("127.0.0.1:%d", c.apiPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return conn, callCtx, cancel, nil
}

// Keep the imports aligned for API operations.
var _ *xtcore.InboundHandlerConfig
var _ *commonprotocol.User
