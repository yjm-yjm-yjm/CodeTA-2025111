package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	pb "template-mall/TemplateWebServer/api/gen/templateorder/v1"
	"template-mall/TemplateWebServer/internal/auth"
	"template-mall/TemplateWebServer/internal/grpcclient"
	"template-mall/TemplateWebServer/internal/middleware"
	"template-mall/TemplateWebServer/internal/response"

	"github.com/gin-gonic/gin"
)

type API struct {
	store  *auth.Store
	tokens *auth.TokenIssuer
	order  *grpcclient.Client
}

func New(store *auth.Store, tokens *auth.TokenIssuer, order *grpcclient.Client) *API {
	return &API{store: store, tokens: tokens, order: order}
}

func (a *API) Register(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	r.POST("/api/auth/register", a.register)
	r.POST("/api/auth/login", a.login)
	// 封面给 <img> 用，不能带 Authorization，故对已上架模板公开
	r.GET("/api/templates/:template_id/cover", a.templateCover)

	authz := r.Group("/api", middleware.JWTAuth(a.tokens))
	authz.GET("/me", a.me)
	authz.GET("/templates", a.listTemplates)
	authz.POST("/templates/:template_id/download", a.download)
	authz.GET("/orders", a.listOrders)
	authz.POST("/orders/:order_id/cancel", a.cancelOrder)
	authz.GET("/membership/plans", a.listMembershipPlans)
	authz.POST("/membership/subscribe", a.subscribeMembership)
}

type registerBody struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

func (a *API) register(c *gin.Context) {
	var body registerBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := a.store.Register(body.Phone, body.Password, body.Nickname)
	if errors.Is(err, auth.ErrUserExists) {
		response.Fail(c, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, auth.ErrInvalidPhone) || errors.Is(err, auth.ErrWeakPassword) {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, auth.ErrInvalidCredential) {
		response.Fail(c, http.StatusBadRequest, "phone and password required")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := a.order.UpsertUser(c.Request.Context(), u.UserID, u.Nickname); err != nil {
		response.FromGRPC(c, err)
		return
	}
	token, err := a.tokens.Issue(u.UserID, u.Phone())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"token": token,
		"user":  gin.H{"user_id": u.UserID, "phone": u.Phone(), "nickname": u.Nickname},
	})
}

type loginBody struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (a *API) login(c *gin.Context) {
	var body loginBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := a.store.Authenticate(body.Phone, body.Password)
	if errors.Is(err, auth.ErrInvalidPhone) {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	if _, err := a.order.UpsertUser(c.Request.Context(), u.UserID, u.Nickname); err != nil {
		response.FromGRPC(c, err)
		return
	}
	token, err := a.tokens.Issue(u.UserID, u.Phone())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"token": token,
		"user":  gin.H{"user_id": u.UserID, "phone": u.Phone(), "nickname": u.Nickname},
	})
}

func (a *API) me(c *gin.Context) {
	uid := middleware.UserID(c)
	local, ok := a.store.GetByUserID(uid)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "user not found")
		return
	}
	biz, err := a.order.GetUser(c.Request.Context(), uid)
	if err != nil {
		// 业务用户可能尚未同步时兜底
		response.OK(c, gin.H{
			"user_id":   local.UserID,
			"phone":     local.Phone(),
			"nickname":  local.Nickname,
			"is_member": false,
		})
		return
	}
	response.OK(c, gin.H{
		"user_id":   local.UserID,
		"phone":     local.Phone(),
		"nickname":  local.Nickname,
		"is_member": biz.GetIsMember(),
	})
}

func (a *API) listTemplates(c *gin.Context) {
	page, size := pageQuery(c)
	fileType := c.Query("file_type")
	resp, err := a.order.ListPublishedTemplates(c.Request.Context(), page, size, fileType)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	items := make([]gin.H, 0, len(resp.GetTemplates()))
	for _, t := range resp.GetTemplates() {
		items = append(items, templateJSON(c, t))
	}
	response.OK(c, gin.H{
		"items": items,
		"page":  pageJSON(resp.GetPage()),
	})
}

func (a *API) templateCover(c *gin.Context) {
	tid := c.Param("template_id")
	if tid == "" {
		response.Fail(c, http.StatusBadRequest, "template_id required")
		return
	}
	tpl, err := a.order.GetTemplate(c.Request.Context(), tid)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	if tpl == nil || !strings.Contains(tpl.GetStatus().String(), "ON_SHELF") {
		response.Fail(c, http.StatusNotFound, "cover not found")
		return
	}
	if url := strings.TrimSpace(tpl.GetCoverUrl()); url != "" {
		c.Redirect(http.StatusFound, url)
		return
	}
	response.Fail(c, http.StatusNotFound, "cover not found")
}

func (a *API) download(c *gin.Context) {
	uid := middleware.UserID(c)
	tid := c.Param("template_id")
	if tid == "" {
		response.Fail(c, http.StatusBadRequest, "template_id required")
		return
	}
	resp, err := a.order.DownloadTemplate(c.Request.Context(), uid, tid)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	if g := resp.GetGranted(); g != nil {
		response.OK(c, gin.H{
			"result":          "granted",
			"order_id":        g.GetOrderId(),
			"download_url":    g.GetDownloadUrl(),
			"expire_at_unix":  g.GetExpireAtUnix(),
			"order":           orderJSON(g.GetOrder(), ""),
		})
		return
	}
	if p := resp.GetPaymentRequired(); p != nil {
		pay := p.GetPayment()
		response.OK(c, gin.H{
			"result": "payment_required",
			"order":  orderJSON(p.GetOrder(), ""),
			"payment": gin.H{
				"pay_order_id": pay.GetPayOrderId(),
				"order_id":     pay.GetOrderId(),
				"amount_fen":   pay.GetAmountFen(),
				"pay_url":      pay.GetPayUrl(),
				"status":       pay.GetStatus(),
			},
		})
		return
	}
	response.Fail(c, http.StatusBadGateway, "empty download result")
}

func (a *API) listMembershipPlans(c *gin.Context) {
	response.OK(c, gin.H{
		"items": []gin.H{
			{"plan": "month", "name": "包月会员", "price_fen": 1500, "desc": "开通后可免费下载付费模板"},
			{"plan": "quarter", "name": "包季会员", "price_fen": 4000, "desc": "开通后可免费下载付费模板"},
			{"plan": "year", "name": "包年会员", "price_fen": 13800, "desc": "开通后可免费下载付费模板"},
		},
	})
}

type subscribeBody struct {
	Plan string `json:"plan"`
}

func (a *API) subscribeMembership(c *gin.Context) {
	uid := middleware.UserID(c)
	var body subscribeBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Plan == "" {
		response.Fail(c, http.StatusBadRequest, "plan required (month|quarter|year)")
		return
	}
	res, err := a.order.SubscribeMembership(c.Request.Context(), uid, body.Plan)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, gin.H{
		"order": orderJSON(res.GetOrder(), membershipName(res.GetOrder().GetTemplateId())),
		"payment": gin.H{
			"pay_order_id": res.GetPayment().GetPayOrderId(),
			"order_id":     res.GetPayment().GetOrderId(),
			"amount_fen":   res.GetPayment().GetAmountFen(),
			"pay_url":      res.GetPayment().GetPayUrl(),
			"status":       res.GetPayment().GetStatus(),
		},
	})
}

func (a *API) listOrders(c *gin.Context) {
	uid := middleware.UserID(c)
	page, size := pageQuery(c)
	resp, err := a.order.ListMyOrders(c.Request.Context(), uid, page, size)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	nameCache := map[string]string{}
	items := make([]gin.H, 0, len(resp.GetOrders()))
	for _, o := range resp.GetOrders() {
		name := membershipName(o.GetTemplateId())
		if name == "" {
			if tid := o.GetTemplateId(); tid != "" {
				if cached, ok := nameCache[tid]; ok {
					name = cached
				} else if tpl, err := a.order.GetTemplate(c.Request.Context(), tid); err == nil && tpl != nil {
					name = tpl.GetName()
					nameCache[tid] = name
				}
			}
		}
		items = append(items, orderJSON(o, name))
	}
	response.OK(c, gin.H{"items": items, "page": pageJSON(resp.GetPage())})
}

func (a *API) cancelOrder(c *gin.Context) {
	uid := middleware.UserID(c)
	oid := c.Param("order_id")
	o, err := a.order.CancelOrder(c.Request.Context(), uid, oid)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	name := membershipName(o.GetTemplateId())
	if name == "" {
		if tpl, err := a.order.GetTemplate(c.Request.Context(), o.GetTemplateId()); err == nil && tpl != nil {
			name = tpl.GetName()
		}
	}
	response.OK(c, orderJSON(o, name))
}

func pageQuery(c *gin.Context) (int32, int32) {
	page := int32(1)
	size := int32(20)
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && v > 0 {
		page = int32(v)
	}
	if v, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && v > 0 {
		size = int32(v)
	}
	return page, size
}

func templateJSON(c *gin.Context, t *pb.Template) gin.H {
	if t == nil {
		return nil
	}
	coverURL := ""
	if strings.TrimSpace(t.GetCoverObjectKey()) != "" || strings.TrimSpace(t.GetCoverUrl()) != "" {
		coverURL = publicAPIURL(c, "/api/templates/"+t.GetTemplateId()+"/cover")
	}
	return gin.H{
		"template_id": t.GetTemplateId(),
		"name":        t.GetName(),
		"file_type":   t.GetFileType(),
		"price_fen":   t.GetPriceFen(),
		"price_type":  t.GetPriceType().String(),
		"status":      t.GetStatus().String(),
		"cover_url":   coverURL,
	}
}

func publicAPIURL(c *gin.Context, path string) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := c.Request.Host
	if host == "" {
		host = "127.0.0.1:8080"
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

func membershipName(templateID string) string {
	switch templateID {
	case "membership_month":
		return "包月会员"
	case "membership_quarter":
		return "包季会员"
	case "membership_year":
		return "包年会员"
	default:
		return ""
	}
}

func orderJSON(o *pb.Order, templateName string) gin.H {
	if o == nil {
		return nil
	}
	if templateName == "" {
		templateName = o.GetTemplateId()
	}
	return gin.H{
		"order_id":           o.GetOrderId(),
		"user_id":            o.GetUserId(),
		"template_id":        o.GetTemplateId(),
		"template_name":      templateName,
		"order_type":         o.GetOrderType().String(),
		"status":             o.GetStatus().String(),
		"price_fen_snapshot": o.GetPriceFenSnapshot(),
		"pay_order_id":       o.GetPayOrderId(),
		"pay_url":            o.GetPayUrl(),
		"created_at_unix":    o.GetCreatedAtUnix(),
		"updated_at_unix":    o.GetUpdatedAtUnix(),
	}
}

func pageJSON(p *pb.PageInfo) gin.H {
	if p == nil {
		return gin.H{"page": 1, "page_size": 20, "total": 0}
	}
	return gin.H{"page": p.GetPage(), "page_size": p.GetPageSize(), "total": p.GetTotal()}
}
