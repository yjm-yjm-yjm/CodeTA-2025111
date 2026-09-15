package grpcclient

import (
	"context"
	"fmt"
	"time"

	pb "template-mall/TemplateWebServer/api/gen/templateorder/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn    *grpc.ClientConn
	order   pb.TemplateOrderServiceClient
	timeout time.Duration
}

func Dial(addr string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial order grpc: %w", err)
	}
	return &Client{
		conn:    conn,
		order:   pb.NewTemplateOrderServiceClient(conn),
		timeout: timeout,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) ctx(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, c.timeout)
}

func (c *Client) UpsertUser(ctx context.Context, userID, nickname string) (*pb.User, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.UpsertUser(ctx, &pb.UpsertUserRequest{UserId: userID, Nickname: nickname})
	if err != nil {
		return nil, err
	}
	return resp.GetUser(), nil
}

func (c *Client) GetUser(ctx context.Context, userID string) (*pb.User, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.GetUser(ctx, &pb.GetUserRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	return resp.GetUser(), nil
}

func (c *Client) ListPublishedTemplates(ctx context.Context, page, pageSize int32, fileType string) (*pb.ListTemplatesResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.order.ListTemplates(ctx, &pb.ListTemplatesRequest{
		Scope:    pb.TemplateListScope_TEMPLATE_LIST_SCOPE_PUBLISHED,
		Page:     &pb.PageRequest{Page: page, PageSize: pageSize},
		FileType: fileType,
	})
}

func (c *Client) DownloadTemplate(ctx context.Context, userID, templateID string) (*pb.DownloadTemplateResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.order.DownloadTemplate(ctx, &pb.DownloadTemplateRequest{
		UserId:     userID,
		TemplateId: templateID,
	})
}

func (c *Client) GetTemplate(ctx context.Context, templateID string) (*pb.Template, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.GetTemplate(ctx, &pb.GetTemplateRequest{TemplateId: templateID})
	if err != nil {
		return nil, err
	}
	return resp.GetTemplate(), nil
}

func (c *Client) SubscribeMembership(ctx context.Context, userID, plan string) (*pb.SubscribeMembershipResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.order.SubscribeMembership(ctx, &pb.SubscribeMembershipRequest{
		UserId: userID,
		Plan:   plan,
	})
}

func (c *Client) ListMyOrders(ctx context.Context, userID string, page, pageSize int32) (*pb.ListOrdersResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.order.ListOrders(ctx, &pb.ListOrdersRequest{
		UserId: userID,
		Page:   &pb.PageRequest{Page: page, PageSize: pageSize},
	})
}

func (c *Client) CancelOrder(ctx context.Context, userID, orderID string) (*pb.Order, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.CancelOrder(ctx, &pb.CancelOrderRequest{
		UserId:  userID,
		OrderId: orderID,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetOrder(), nil
}
