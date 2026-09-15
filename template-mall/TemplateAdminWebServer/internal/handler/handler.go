package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "template-mall/TemplateAdminWebServer/api/gen/templateorder/v1"
	"template-mall/TemplateAdminWebServer/internal/auth"
	"template-mall/TemplateAdminWebServer/internal/covergen"
	"template-mall/TemplateAdminWebServer/internal/grpcclient"
	"template-mall/TemplateAdminWebServer/internal/middleware"
	"template-mall/TemplateAdminWebServer/internal/ossstore"
	"template-mall/TemplateAdminWebServer/internal/response"
	"template-mall/TemplateAdminWebServer/internal/wpsoauth"

	"github.com/gin-gonic/gin"
)

type API struct {
	mode        string // mock | wps
	tokens      *auth.TokenIssuer
	oauth       *wpsoauth.Client
	frontendURL string
	order       *grpcclient.Client
	oss         ossstore.Store

	stateMu sync.Mutex
	states  map[string]time.Time
}

func New(mode string, tokens *auth.TokenIssuer, oauth *wpsoauth.Client, frontendURL string, order *grpcclient.Client, oss ossstore.Store) *API {
	return &API{
		mode: mode, tokens: tokens, oauth: oauth, frontendURL: frontendURL,
		order: order, oss: oss, states: map[string]time.Time{},
	}
}

func (a *API) Register(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	r.GET("/api/auth/login", a.login)
	r.GET("/api/auth/callback", a.callback)
	r.POST("/api/auth/mock-login", a.mockLogin)
	r.POST("/api/auth/logout", a.logout)

	// mock OSS 鐩翠紶鍏ュ彛锛堜粎 OSS_PROVIDER=mock锛?
	r.PUT("/api/upload/mock", a.mockUpload)
	// 灏侀潰缁?<img> 鐢紱绠＄悊绔垪琛ㄤ篃璧板悓婧愪唬鐞?
	r.GET("/api/templates/:template_id/cover", a.templateCover)

	admin := r.Group("/api", middleware.AdminJWT(a.tokens))
	admin.GET("/me", a.me)
	admin.POST("/upload/credential", a.uploadCredential)
	admin.POST("/upload/confirm", a.uploadConfirm)
	admin.POST("/templates", a.createTemplate)
	admin.PATCH("/templates/:template_id", a.updateTemplate)
	admin.POST("/templates/:template_id/publish", a.publishTemplate)
	admin.POST("/templates/:template_id/unpublish", a.unpublishTemplate)
	admin.GET("/templates", a.listTemplates)
	admin.GET("/templates/:template_id", a.getTemplate)
	admin.GET("/orders", a.listOrders)
	admin.POST("/users/:user_id/membership", a.setMembership)
}

func (a *API) login(c *gin.Context) {
	if a.mode != "wps" {
		response.OK(c, gin.H{
			"mode":    "mock",
			"message": "AUTH_MODE=mock锛屽綋鍓嶆湭鍚敤 WPS OAuth",
		})
		return
	}
	state := randomState()
	a.stateMu.Lock()
	a.states[state] = time.Now().Add(10 * time.Minute)
	a.stateMu.Unlock()
	authorizeURL := a.oauth.AuthCodeURL(state)
	// SPA / Vite 浠ｇ悊鍦烘櫙锛氳繑鍥炵粷瀵规巿鏉冨湴鍧€锛岀敱娴忚鍣ㄧ洿鎺ヨ烦 WPS锛堥伩鍏嶄唬鐞嗗悶鎺?302锛?
	if c.Query("format") == "json" || prefersJSON(c) {
		response.OK(c, gin.H{
			"mode":          "wps",
			"authorize_url": authorizeURL,
		})
		return
	}
	c.Redirect(http.StatusFound, authorizeURL)
}

func prefersJSON(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "application/json")
}

func (a *API) callback(c *gin.Context) {
	if a.mode != "wps" {
		response.Fail(c, http.StatusBadRequest, "wps auth disabled")
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		response.Fail(c, http.StatusBadRequest, "missing code/state")
		return
	}
	a.stateMu.Lock()
	exp, ok := a.states[state]
	delete(a.states, state)
	a.stateMu.Unlock()
	if !ok || time.Now().After(exp) {
		response.Fail(c, http.StatusBadRequest, "invalid state")
		return
	}

	tok, err := a.oauth.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	user, err := a.oauth.GetCurrentUser(c.Request.Context(), tok.AccessToken)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	// 姝ｅ紡鐜锛氱櫥褰曞嵆绠＄悊鍛?
	jwtTok, err := a.tokens.Issue(user.ID, user.Nickname, "wps")
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	redir, _ := url.Parse(a.frontendURL)
	q := redir.Query()
	q.Set("token", jwtTok)
	redir.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, redir.String())
}

type mockLoginBody struct {
	AdminID  string `json:"admin_id"`
	Nickname string `json:"nickname"`
}

func (a *API) mockLogin(c *gin.Context) {
	// 鏈湴鑱旇皟锛氬嵆浣?AUTH_MODE=wps 涔熶繚鐣?mock锛屼笌 WPS 骞跺瓨
	var body mockLoginBody
	_ = c.ShouldBindJSON(&body)
	if body.AdminID == "" {
		body.AdminID = "mock-admin"
	}
	if body.Nickname == "" {
		body.Nickname = "MockAdmin"
	}
	tok, err := a.tokens.Issue(body.AdminID, body.Nickname, "mock")
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"token": tok,
		"admin": gin.H{"admin_id": body.AdminID, "nickname": body.Nickname},
	})
}

func (a *API) logout(c *gin.Context) {
	response.OK(c, gin.H{"ok": true})
}

func (a *API) me(c *gin.Context) {
	response.OK(c, gin.H{
		"admin_id": middleware.AdminID(c),
		"nickname": c.GetString(middleware.CtxNickname),
		"role":     "admin",
	})
}

type credentialBody struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

func (a *API) uploadCredential(c *gin.Context) {
	var body credentialBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Filename == "" {
		response.Fail(c, http.StatusBadRequest, "filename required")
		return
	}
	cred, err := a.oss.IssueUpload(c.Request.Context(), body.Filename, body.ContentType)
	if err != nil {
		writeOSSErr(c, err)
		return
	}
	response.OK(c, cred)
}

func (a *API) mockUpload(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, http.StatusBadRequest, "key required")
		return
	}
	ms, ok := a.oss.(*ossstore.MockStore)
	if !ok {
		response.Fail(c, http.StatusNotFound, "mock upload disabled")
		return
	}
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, ossstore.MaxFileSize+1))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if int64(len(data)) > ossstore.MaxFileSize {
		response.Fail(c, http.StatusBadRequest, "file too large")
		return
	}
	if err := ms.PutMockObject(key, int64(len(data))); err != nil {
		writeOSSErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type confirmBody struct {
	ObjectKey        string `json:"object_key"`
	CoverObjectKey   string `json:"cover_object_key"`
	OriginalFilename string `json:"original_filename"`
	FileType         string `json:"file_type"`
	FileSize         int64  `json:"file_size"`
	Name             string `json:"name"`
	PriceFen         int64  `json:"price_fen"`
	PriceType        string `json:"price_type"` // free|paid
	Publish          bool   `json:"publish"`
}

// uploadConfirm 纭瀵硅薄瀛樺湪鍚庡垱寤烘ā鏉匡紙涓€姝ュ畬鎴愪綔涓氶摼璺級
func (a *API) uploadConfirm(c *gin.Context) {
	var body confirmBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid json")
		return
	}
	if body.ObjectKey == "" || body.OriginalFilename == "" || body.Name == "" {
		response.Fail(c, http.StatusBadRequest, "object_key/original_filename/name required")
		return
	}
	meta, err := a.oss.Confirm(c.Request.Context(), body.ObjectKey, body.FileSize)
	if err != nil {
		writeOSSErr(c, err)
		return
	}
	priceType := pb.PriceType_PRICE_TYPE_FREE
	if body.PriceType == "paid" || body.PriceFen > 0 {
		priceType = pb.PriceType_PRICE_TYPE_PAID
	}
	fileType := body.FileType
	if fileType == "" {
		fileType = strings.TrimPrefix(strings.ToLower(pathExt(body.OriginalFilename)), ".")
	}
	coverKey := body.CoverObjectKey
	if coverKey != "" {
		if _, err := a.oss.Confirm(c.Request.Context(), coverKey, 0); err != nil {
			writeOSSErr(c, err)
			return
		}
	} else if key, err := a.autoCoverFromObject(c, meta.Key, fileType); err == nil {
		coverKey = key
	}
	tpl, err := a.order.CreateTemplate(c.Request.Context(), &pb.CreateTemplateRequest{
		Name:             body.Name,
		FileType:         fileType,
		PriceFen:         body.PriceFen,
		PriceType:        priceType,
		StorageProvider:  a.oss.ProviderName(),
		Bucket:           meta.Bucket,
		ObjectKey:        meta.Key,
		CoverObjectKey:   coverKey,
		OriginalFilename: body.OriginalFilename,
		FileSize:         meta.Size,
		Publish:          body.Publish,
	})
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, templateJSON(c, tpl))
}

type createTemplateBody struct {
	Name             string `json:"name"`
	FileType         string `json:"file_type"`
	PriceFen         int64  `json:"price_fen"`
	PriceType        string `json:"price_type"`
	ObjectKey        string `json:"object_key"`
	CoverObjectKey   string `json:"cover_object_key"`
	OriginalFilename string `json:"original_filename"`
	FileSize         int64  `json:"file_size"`
	Publish          bool   `json:"publish"`
}

func (a *API) createTemplate(c *gin.Context) {
	var body createTemplateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid json")
		return
	}
	meta, err := a.oss.Confirm(c.Request.Context(), body.ObjectKey, body.FileSize)
	if err != nil {
		writeOSSErr(c, err)
		return
	}
	priceType := pb.PriceType_PRICE_TYPE_FREE
	if body.PriceType == "paid" || body.PriceFen > 0 {
		priceType = pb.PriceType_PRICE_TYPE_PAID
	}
	tpl, err := a.order.CreateTemplate(c.Request.Context(), &pb.CreateTemplateRequest{
		Name: body.Name, FileType: body.FileType, PriceFen: body.PriceFen, PriceType: priceType,
		StorageProvider: a.oss.ProviderName(), Bucket: meta.Bucket, ObjectKey: meta.Key,
		CoverObjectKey: body.CoverObjectKey, OriginalFilename: body.OriginalFilename, FileSize: meta.Size, Publish: body.Publish,
	})
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, templateJSON(c, tpl))
}

type updateTemplateBody struct {
	Name      *string `json:"name"`
	PriceFen  *int64  `json:"price_fen"`
	PriceType *string `json:"price_type"`
}

func (a *API) updateTemplate(c *gin.Context) {
	var body updateTemplateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid json")
		return
	}
	req := &pb.UpdateTemplateRequest{TemplateId: c.Param("template_id")}
	if body.Name != nil {
		req.Name = body.Name
	}
	if body.PriceFen != nil {
		req.PriceFen = body.PriceFen
	}
	if body.PriceType != nil {
		pt := pb.PriceType_PRICE_TYPE_UNSPECIFIED
		switch *body.PriceType {
		case "free":
			pt = pb.PriceType_PRICE_TYPE_FREE
		case "paid":
			pt = pb.PriceType_PRICE_TYPE_PAID
		}
		req.PriceType = &pt
	}
	tpl, err := a.order.UpdateTemplate(c.Request.Context(), req)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, templateJSON(c, tpl))
}

func (a *API) publishTemplate(c *gin.Context) {
	tpl, err := a.order.SetTemplateShelf(c.Request.Context(), c.Param("template_id"), true)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, templateJSON(c, tpl))
}

func (a *API) unpublishTemplate(c *gin.Context) {
	tpl, err := a.order.SetTemplateShelf(c.Request.Context(), c.Param("template_id"), false)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, templateJSON(c, tpl))
}

func (a *API) listTemplates(c *gin.Context) {
	page, size := pageQuery(c)
	resp, err := a.order.ListTemplatesAdmin(c.Request.Context(), page, size, c.Query("file_type"))
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	items := make([]gin.H, 0, len(resp.GetTemplates()))
	for _, t := range resp.GetTemplates() {
		items = append(items, templateJSON(c, t))
	}
	response.OK(c, gin.H{"items": items, "page": pageJSON(resp.GetPage())})
}

func (a *API) getTemplate(c *gin.Context) {
	tpl, err := a.order.GetTemplate(c.Request.Context(), c.Param("template_id"))
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, templateJSON(c, tpl))
}

func (a *API) listOrders(c *gin.Context) {
	page, size := pageQuery(c)
	resp, err := a.order.ListOrders(c.Request.Context(), page, size)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	type userInfo struct {
		name     string
		isMember bool
	}
	userCache := map[string]userInfo{}
	tplCache := map[string]string{}
	items := make([]gin.H, 0, len(resp.GetOrders()))
	for _, o := range resp.GetOrders() {
		ui := userInfo{}
		if uid := o.GetUserId(); uid != "" {
			if cached, ok := userCache[uid]; ok {
				ui = cached
			} else if u, err := a.order.GetUser(c.Request.Context(), uid); err == nil && u != nil {
				ui = userInfo{name: u.GetNickname(), isMember: u.GetIsMember()}
				userCache[uid] = ui
			}
		}
		tplName := membershipTemplateName(o.GetTemplateId())
		if tplName == "" {
			if tid := o.GetTemplateId(); tid != "" {
				if cached, ok := tplCache[tid]; ok {
					tplName = cached
				} else if tpl, err := a.order.GetTemplate(c.Request.Context(), tid); err == nil && tpl != nil {
					tplName = tpl.GetName()
					tplCache[tid] = tplName
				}
			}
		}
		items = append(items, orderJSON(o, ui.name, tplName, ui.isMember))
	}
	response.OK(c, gin.H{"items": items, "page": pageJSON(resp.GetPage())})
}

type membershipBody struct {
	IsMember bool `json:"is_member"`
}

func (a *API) setMembership(c *gin.Context) {
	var body membershipBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := a.order.SetMembership(c.Request.Context(), c.Param("user_id"), body.IsMember)
	if err != nil {
		response.FromGRPC(c, err)
		return
	}
	response.OK(c, gin.H{
		"user_id": u.GetUserId(), "nickname": u.GetNickname(), "is_member": u.GetIsMember(),
	})
}

func writeOSSErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ossstore.ErrInvalidArgument), errors.Is(err, ossstore.ErrObjectTooLarge), errors.Is(err, ossstore.ErrBadObjectKey):
		response.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ossstore.ErrObjectNotFound):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, err.Error())
	}
}

func pageQuery(c *gin.Context) (int32, int32) {
	page, size := int32(1), int32(20)
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
		"template_id": t.GetTemplateId(), "name": t.GetName(), "file_type": t.GetFileType(),
		"price_fen": t.GetPriceFen(), "price_type": t.GetPriceType().String(), "status": t.GetStatus().String(),
		"storage_provider": t.GetStorageProvider(), "bucket": t.GetBucket(), "object_key": t.GetObjectKey(),
		"cover_object_key": t.GetCoverObjectKey(), "cover_url": coverURL,
		"original_filename": t.GetOriginalFilename(), "file_size": t.GetFileSize(),
	}
}

func publicAPIURL(c *gin.Context, p string) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := c.Request.Host
	if host == "" {
		host = "127.0.0.1:8081"
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, p)
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
	if tpl == nil {
		response.Fail(c, http.StatusNotFound, "cover not found")
		return
	}
	if u := strings.TrimSpace(tpl.GetCoverUrl()); u != "" {
		c.Redirect(http.StatusFound, u)
		return
	}
	response.Fail(c, http.StatusNotFound, "cover not found")
}

func (a *API) autoCoverFromObject(c *gin.Context, objectKey, fileType string) (string, error) {
	ft := strings.ToLower(strings.TrimPrefix(fileType, "."))
	if ft != "pptx" && ft != "docx" {
		return "", fmt.Errorf("auto cover unsupported for %s", ft)
	}
	rc, err := a.oss.GetObject(c.Request.Context(), objectKey)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	ra, size, err := covergen.EnsureReaderAt(rc)
	if err != nil {
		return "", err
	}
	ct, data, err := covergen.FromOfficeZIP(ft, ra, size)
	if err != nil {
		return "", err
	}
	ext := ".png"
	switch ct {
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	}
	key := a.oss.NewObjectKey(ext)
	if err := a.oss.PutObject(c.Request.Context(), key, ct, bytes.NewReader(data), int64(len(data))); err != nil {
		return "", err
	}
	return key, nil
}

func orderJSON(o *pb.Order, userName, templateName string, isMember bool) gin.H {
	if o == nil {
		return nil
	}
	return gin.H{
		"order_id":           o.GetOrderId(),
		"user_id":            o.GetUserId(),
		"user_name":          userName,
		"template_id":        o.GetTemplateId(),
		"template_name":      templateName,
		"is_member":          isMember,
		"order_type":         o.GetOrderType().String(),
		"status":             o.GetStatus().String(),
		"price_fen_snapshot": o.GetPriceFenSnapshot(),
		"pay_order_id":       o.GetPayOrderId(),
		"pay_url":            o.GetPayUrl(),
	}
}

func membershipTemplateName(templateID string) string {
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

func pageJSON(p *pb.PageInfo) gin.H {
	if p == nil {
		return gin.H{"page": 1, "page_size": 20, "total": 0}
	}
	return gin.H{"page": p.GetPage(), "page_size": p.GetPageSize(), "total": p.GetTotal()}
}

func pathExt(name string) string {
	i := strings.LastIndex(name, ".")
	if i < 0 {
		return ""
	}
	return name[i:]
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

