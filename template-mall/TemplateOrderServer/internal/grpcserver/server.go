package grpcserver

import (
	"context"
	"errors"

	pb "template-mall/TemplateOrderServer/api/gen/templateorder/v1"
	"template-mall/TemplateOrderServer/internal/model"
	"template-mall/TemplateOrderServer/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedTemplateOrderServiceServer
	svc *service.Service
}

func New(svc *service.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) UpsertUser(ctx context.Context, req *pb.UpsertUserRequest) (*pb.UpsertUserResponse, error) {
	u, err := s.svc.UpsertUser(ctx, req.GetUserId(), req.GetNickname())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.UpsertUserResponse{User: toPBUser(u)}, nil
}

func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	u, err := s.svc.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.GetUserResponse{User: toPBUser(u)}, nil
}

func (s *Server) SetMembership(ctx context.Context, req *pb.SetMembershipRequest) (*pb.SetMembershipResponse, error) {
	u, err := s.svc.SetMembership(ctx, req.GetUserId(), req.GetIsMember())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.SetMembershipResponse{User: toPBUser(u)}, nil
}

func (s *Server) SubscribeMembership(ctx context.Context, req *pb.SubscribeMembershipRequest) (*pb.SubscribeMembershipResponse, error) {
	res, err := s.svc.SubscribeMembership(ctx, req.GetUserId(), req.GetPlan())
	if err != nil {
		return nil, mapErr(err)
	}
	payStatus := res.Payment.Status
	if payStatus == "" {
		payStatus = "unpaid"
	}
	return &pb.SubscribeMembershipResponse{
		Order: toPBOrder(res.Order),
		Payment: &pb.PaymentInfo{
			PayOrderId: res.Payment.PayOrderID,
			OrderId:    res.Payment.OrderID,
			AmountFen:  res.Payment.AmountFen,
			PayUrl:     res.Payment.PayURL,
			Status:     payStatus,
		},
	}, nil
}

func (s *Server) CreateTemplate(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.CreateTemplateResponse, error) {
	t, err := s.svc.CreateTemplate(ctx, service.CreateTemplateInput{
		Name:             req.GetName(),
		FileType:         req.GetFileType(),
		PriceFen:         req.GetPriceFen(),
		PriceType:        fromPBPriceType(req.GetPriceType()),
		StorageProvider:  req.GetStorageProvider(),
		Bucket:           req.GetBucket(),
		ObjectKey:        req.GetObjectKey(),
		CoverObjectKey:   req.GetCoverObjectKey(),
		OriginalFilename: req.GetOriginalFilename(),
		FileSize:         req.GetFileSize(),
		Publish:          req.GetPublish(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.CreateTemplateResponse{Template: s.toPBTemplate(ctx, t)}, nil
}

func (s *Server) UpdateTemplate(ctx context.Context, req *pb.UpdateTemplateRequest) (*pb.UpdateTemplateResponse, error) {
	var name *string
	var priceFen *int64
	var priceType *string
	if req.Name != nil {
		v := req.GetName()
		name = &v
	}
	if req.PriceFen != nil {
		v := req.GetPriceFen()
		priceFen = &v
	}
	if req.PriceType != nil {
		v := fromPBPriceType(req.GetPriceType())
		priceType = &v
	}
	t, err := s.svc.UpdateTemplate(ctx, req.GetTemplateId(), name, priceFen, priceType)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.UpdateTemplateResponse{Template: s.toPBTemplate(ctx, t)}, nil
}

func (s *Server) SetTemplateShelf(ctx context.Context, req *pb.SetTemplateShelfRequest) (*pb.SetTemplateShelfResponse, error) {
	publish := req.GetAction() == pb.ShelfAction_SHELF_ACTION_PUBLISH
	if req.GetAction() == pb.ShelfAction_SHELF_ACTION_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "shelf action required")
	}
	t, err := s.svc.SetTemplateShelf(ctx, req.GetTemplateId(), publish)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.SetTemplateShelfResponse{Template: s.toPBTemplate(ctx, t)}, nil
}

func (s *Server) GetTemplate(ctx context.Context, req *pb.GetTemplateRequest) (*pb.GetTemplateResponse, error) {
	t, err := s.svc.GetTemplate(ctx, req.GetTemplateId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.GetTemplateResponse{Template: s.toPBTemplate(ctx, t)}, nil
}

func (s *Server) ListTemplates(ctx context.Context, req *pb.ListTemplatesRequest) (*pb.ListTemplatesResponse, error) {
	onlyOnShelf := req.GetScope() != pb.TemplateListScope_TEMPLATE_LIST_SCOPE_ADMIN
	page, size := pageParams(req.GetPage())
	list, total, err := s.svc.ListTemplates(ctx, onlyOnShelf, req.GetFileType(), page, size)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*pb.Template, 0, len(list))
	for i := range list {
		out = append(out, s.toPBTemplate(ctx, &list[i]))
	}
	return &pb.ListTemplatesResponse{
		Templates: out,
		Page:      &pb.PageInfo{Page: int32(page), PageSize: int32(size), Total: total},
	}, nil
}

func (s *Server) DownloadTemplate(ctx context.Context, req *pb.DownloadTemplateRequest) (*pb.DownloadTemplateResponse, error) {
	res, err := s.svc.DownloadTemplate(ctx, req.GetUserId(), req.GetTemplateId())
	if err != nil {
		return nil, mapErr(err)
	}
	if res.Granted != nil {
		return &pb.DownloadTemplateResponse{
			Result: &pb.DownloadTemplateResponse_Granted{
				Granted: &pb.DownloadGranted{
					OrderId:       res.Granted.Order.OrderID,
					DownloadUrl:   res.Granted.URL,
					ExpireAtUnix:  res.Granted.ExpireAt.Unix(),
					Order:         toPBOrder(res.Granted.Order),
				},
			},
		}, nil
	}
	return &pb.DownloadTemplateResponse{
		Result: &pb.DownloadTemplateResponse_PaymentRequired{
			PaymentRequired: &pb.PaymentRequired{
				Order: toPBOrder(res.NeedPay.Order),
				Payment: &pb.PaymentInfo{
					PayOrderId: res.NeedPay.Payment.PayOrderID,
					OrderId:    res.NeedPay.Payment.OrderID,
					AmountFen:  res.NeedPay.Payment.AmountFen,
					PayUrl:     res.NeedPay.Payment.PayURL,
					Status:     res.NeedPay.Payment.Status,
				},
			},
		},
	}, nil
}

func (s *Server) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	page, size := pageParams(req.GetPage())
	list, total, err := s.svc.ListOrders(ctx, req.GetUserId(), fromPBOrderType(req.GetOrderType()), fromPBOrderStatus(req.GetStatus()), page, size)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*pb.Order, 0, len(list))
	for i := range list {
		out = append(out, toPBOrder(&list[i]))
	}
	return &pb.ListOrdersResponse{
		Orders: out,
		Page:   &pb.PageInfo{Page: int32(page), PageSize: int32(size), Total: total},
	}, nil
}

func (s *Server) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	o, err := s.svc.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.GetOrderResponse{Order: toPBOrder(o)}, nil
}

func (s *Server) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	o, err := s.svc.CancelOrder(ctx, req.GetUserId(), req.GetOrderId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.CancelOrderResponse{Order: toPBOrder(o)}, nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, service.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrInvalidArgument), errors.Is(err, service.ErrCancelNotAllowed):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrTemplateOffShelf):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrAmountMismatch):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrOrderCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}

func pageParams(p *pb.PageRequest) (int, int) {
	page, size := 1, 20
	if p != nil {
		if p.Page > 0 {
			page = int(p.Page)
		}
		if p.PageSize > 0 {
			size = int(p.PageSize)
		}
	}
	return page, size
}

func toPBUser(u *model.User) *pb.User {
	return &pb.User{
		UserId:        u.UserID,
		Nickname:      u.Nickname,
		IsMember:      u.IsMember,
		CreatedAtUnix: u.CreatedAt.Unix(),
		UpdatedAtUnix: u.UpdatedAt.Unix(),
	}
}

func toPBTemplate(t *model.Template) *pb.Template {
	return &pb.Template{
		TemplateId:       t.TemplateID,
		Name:             t.Name,
		FileType:         t.FileType,
		PriceFen:         t.PriceFen,
		PriceType:        toPBPriceType(t.PriceType),
		Status:           toPBTemplateStatus(t.Status),
		StorageProvider:  t.StorageProvider,
		Bucket:           t.Bucket,
		ObjectKey:        t.ObjectKey,
		CoverObjectKey:   t.CoverObjectKey,
		OriginalFilename: t.OriginalFilename,
		FileSize:         t.FileSize,
		CreatedAtUnix:    t.CreatedAt.Unix(),
		UpdatedAtUnix:    t.UpdatedAt.Unix(),
	}
}

func (s *Server) toPBTemplate(ctx context.Context, t *model.Template) *pb.Template {
	out := toPBTemplate(t)
	if t.CoverObjectKey == "" {
		return out
	}
	url, err := s.svc.PresignCoverURL(ctx, t)
	if err == nil {
		out.CoverUrl = url
	}
	return out
}

func toPBOrder(o *model.Order) *pb.Order {
	return &pb.Order{
		OrderId:          o.OrderID,
		UserId:           o.UserID,
		TemplateId:       o.TemplateID,
		OrderType:        toPBOrderType(o.OrderType),
		Status:           toPBOrderStatus(o.Status),
		PriceFenSnapshot: o.PriceFenSnapshot,
		PayOrderId:       o.PayOrderID,
		PayUrl:           o.PayURL,
		CreatedAtUnix:    o.CreatedAt.Unix(),
		UpdatedAtUnix:    o.UpdatedAt.Unix(),
	}
}

func toPBPriceType(v string) pb.PriceType {
	switch v {
	case model.PriceTypeFree:
		return pb.PriceType_PRICE_TYPE_FREE
	case model.PriceTypePaid:
		return pb.PriceType_PRICE_TYPE_PAID
	default:
		return pb.PriceType_PRICE_TYPE_UNSPECIFIED
	}
}

func fromPBPriceType(v pb.PriceType) string {
	switch v {
	case pb.PriceType_PRICE_TYPE_FREE:
		return model.PriceTypeFree
	case pb.PriceType_PRICE_TYPE_PAID:
		return model.PriceTypePaid
	default:
		return ""
	}
}

func toPBTemplateStatus(v string) pb.TemplateStatus {
	switch v {
	case model.TemplateStatusDraft:
		return pb.TemplateStatus_TEMPLATE_STATUS_DRAFT
	case model.TemplateStatusOnShelf:
		return pb.TemplateStatus_TEMPLATE_STATUS_ON_SHELF
	case model.TemplateStatusOffShelf:
		return pb.TemplateStatus_TEMPLATE_STATUS_OFF_SHELF
	default:
		return pb.TemplateStatus_TEMPLATE_STATUS_UNSPECIFIED
	}
}

func toPBOrderType(v string) pb.OrderType {
	switch v {
	case model.OrderTypeFree:
		return pb.OrderType_ORDER_TYPE_FREE
	case model.OrderTypeMember:
		return pb.OrderType_ORDER_TYPE_MEMBER
	case model.OrderTypeRetail:
		return pb.OrderType_ORDER_TYPE_RETAIL
	default:
		return pb.OrderType_ORDER_TYPE_UNSPECIFIED
	}
}

func fromPBOrderType(v pb.OrderType) string {
	switch v {
	case pb.OrderType_ORDER_TYPE_FREE:
		return model.OrderTypeFree
	case pb.OrderType_ORDER_TYPE_MEMBER:
		return model.OrderTypeMember
	case pb.OrderType_ORDER_TYPE_RETAIL:
		return model.OrderTypeRetail
	default:
		return ""
	}
}

func toPBOrderStatus(v string) pb.OrderStatus {
	switch v {
	case model.OrderStatusUnpaid:
		return pb.OrderStatus_ORDER_STATUS_UNPAID
	case model.OrderStatusDownloadReady:
		return pb.OrderStatus_ORDER_STATUS_DOWNLOAD_READY
	case model.OrderStatusCancelled:
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func fromPBOrderStatus(v pb.OrderStatus) string {
	switch v {
	case pb.OrderStatus_ORDER_STATUS_UNPAID:
		return model.OrderStatusUnpaid
	case pb.OrderStatus_ORDER_STATUS_DOWNLOAD_READY:
		return model.OrderStatusDownloadReady
	case pb.OrderStatus_ORDER_STATUS_CANCELLED:
		return model.OrderStatusCancelled
	default:
		return ""
	}
}
