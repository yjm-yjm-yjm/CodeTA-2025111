package grpcclient

import (
	"context"
	"fmt"
	"time"

	pb "template-mall/TemplateAdminWebServer/api/gen/templateorder/v1"

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
	return &Client{conn: conn, order: pb.NewTemplateOrderServiceClient(conn), timeout: timeout}, nil
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

func (c *Client) CreateTemplate(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.Template, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.CreateTemplate(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetTemplate(), nil
}

func (c *Client) UpdateTemplate(ctx context.Context, req *pb.UpdateTemplateRequest) (*pb.Template, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.UpdateTemplate(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetTemplate(), nil
}

func (c *Client) SetTemplateShelf(ctx context.Context, templateID string, publish bool) (*pb.Template, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	action := pb.ShelfAction_SHELF_ACTION_UNPUBLISH
	if publish {
		action = pb.ShelfAction_SHELF_ACTION_PUBLISH
	}
	resp, err := c.order.SetTemplateShelf(ctx, &pb.SetTemplateShelfRequest{
		TemplateId: templateID,
		Action:     action,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetTemplate(), nil
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

func (c *Client) GetUser(ctx context.Context, userID string) (*pb.User, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.GetUser(ctx, &pb.GetUserRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	return resp.GetUser(), nil
}

func (c *Client) ListTemplatesAdmin(ctx context.Context, page, size int32, fileType string) (*pb.ListTemplatesResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.order.ListTemplates(ctx, &pb.ListTemplatesRequest{
		Scope:    pb.TemplateListScope_TEMPLATE_LIST_SCOPE_ADMIN,
		Page:     &pb.PageRequest{Page: page, PageSize: size},
		FileType: fileType,
	})
}

func (c *Client) ListOrders(ctx context.Context, page, size int32) (*pb.ListOrdersResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.order.ListOrders(ctx, &pb.ListOrdersRequest{
		Page: &pb.PageRequest{Page: page, PageSize: size},
	})
}

func (c *Client) SetMembership(ctx context.Context, userID string, isMember bool) (*pb.User, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	resp, err := c.order.SetMembership(ctx, &pb.SetMembershipRequest{
		UserId:   userID,
		IsMember: isMember,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetUser(), nil
}
