package handler

import (
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"

	"template-mall/PayWebServer/internal/service"

	"github.com/gin-gonic/gin"
)

type API struct {
	svc             *service.Service
	mallFrontendURL string
}

func New(svc *service.Service, mallFrontendURL string) *API {
	u := strings.TrimRight(mallFrontendURL, "/")
	if u == "" {
		u = "http://127.0.0.1:3000"
	}
	return &API{svc: svc, mallFrontendURL: u}
}

func (a *API) Register(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	r.POST("/api/payments", a.createPayment)
	// 作业固定回调路径，免鉴权；支付成功必须走此入口
	r.POST("/api/paycallback", a.payCallback)

	r.GET("/pay/:pay_order_id", a.payPage)
}

type createPaymentBody struct {
	OrderID   string `json:"order_id"`
	AmountFen int64  `json:"amount_fen"`
	UserID    string `json:"user_id"`
	Subject   string `json:"subject"`
}

func (a *API) createPayment(c *gin.Context) {
	var body createPaymentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := a.svc.CreatePayment(c.Request.Context(), service.CreatePaymentInput{
		OrderID:   body.OrderID,
		AmountFen: body.AmountFen,
		UserID:    body.UserID,
		Subject:   body.Subject,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

type callbackBody struct {
	CallbackID string `json:"callback_id"`
	PayOrderID string `json:"pay_order_id"`
	OrderID    string `json:"order_id"`
	AmountFen  int64  `json:"amount_fen"`
}

func (a *API) payCallback(c *gin.Context) {
	raw, _ := io.ReadAll(c.Request.Body)
	var body callbackBody
	_ = json.Unmarshal(raw, &body)

	// 兼容 form
	if body.PayOrderID == "" {
		body.PayOrderID = c.PostForm("pay_order_id")
	}
	if body.OrderID == "" {
		body.OrderID = c.PostForm("order_id")
	}
	if body.CallbackID == "" {
		body.CallbackID = c.PostForm("callback_id")
	}
	if body.AmountFen == 0 {
		if v := c.PostForm("amount_fen"); v != "" {
			body.AmountFen, _ = strconv.ParseInt(v, 10, 64)
		}
	}

	out, err := a.svc.HandleCallback(c.Request.Context(), service.CallbackInput{
		CallbackID: body.CallbackID,
		PayOrderID: body.PayOrderID,
		OrderID:    body.OrderID,
		AmountFen:  body.AmountFen,
		RawBody:    string(raw),
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (a *API) payPage(c *gin.Context) {
	id := c.Param("pay_order_id")
	o, err := a.svc.GetPayOrder(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = c.Writer.Write([]byte(renderPayHTML(
		o.PayOrderID, o.OrderID, o.AmountFen, o.Status, o.Subject, a.mallFrontendURL,
	)))
}

func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidArgument), errors.Is(err, service.ErrAmountMismatch):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func renderPayHTML(payOrderID, orderID string, amountFen int64, status, subject, mallURL string) string {
	yuan := float64(amountFen) / 100
	callbackID := "mock-" + payOrderID
	paid := status == "paid"
	statusText := "待支付"
	if paid {
		statusText = "已支付"
	}
	successClass := ""
	if paid {
		successClass = "show"
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>模拟支付</title>
<style>
  :root {
    --bg1: #e8f2ef;
    --bg2: #f7f3ea;
    --card: #ffffff;
    --line: #d7e0e6;
    --text: #14212b;
    --muted: #667788;
    --primary: #0f6a5c;
    --primary-hover: #0c574b;
    --ok: #067647;
    --ok-bg: #e8f8ef;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    min-height: 100vh;
    font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
    color: var(--text);
    background:
      radial-gradient(ellipse at top left, var(--bg1), transparent 55%),
      radial-gradient(ellipse at bottom right, var(--bg2), transparent 50%),
      #f3f5f7;
    display: grid;
    place-items: center;
    padding: 24px;
  }
  .card {
    width: min(480px, 100%);
    background: var(--card);
    border: 1px solid var(--line);
    border-radius: 16px;
    padding: 28px 24px 22px;
    box-shadow: 0 16px 40px rgba(20, 33, 43, 0.08);
  }
  h1 {
    margin: 0 0 6px;
    font-size: 1.45rem;
  }
  .sub {
    margin: 0 0 20px;
    color: var(--muted);
    font-size: 0.92rem;
  }
  .field {
    margin-bottom: 12px;
  }
  .field label {
    display: block;
    margin-bottom: 6px;
    font-size: 0.85rem;
    color: var(--muted);
  }
  .field .box {
    width: 100%;
    border: 1px solid var(--line);
    background: #f8fafb;
    border-radius: 10px;
    padding: 11px 12px;
    font-size: 0.98rem;
    word-break: break-all;
  }
  .amount .box {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--primary);
    background: #eef8f5;
    border-color: #b7ddd5;
  }
  .status-ok { color: var(--ok); font-weight: 700; }
  .actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 18px;
  }
  button, .btn {
    appearance: none;
    border: 0;
    border-radius: 10px;
    padding: 12px 16px;
    font: inherit;
    cursor: pointer;
    text-align: center;
    text-decoration: none;
  }
  #btn {
    background: var(--primary);
    color: #fff;
  }
  #btn:hover:not(:disabled) { background: var(--primary-hover); }
  #btn:disabled {
    background: #9aa7b2;
    cursor: not-allowed;
  }
  .btn-back {
    background: #fff;
    color: var(--primary);
    border: 1px solid #b7ddd5;
  }
  .success {
    display: none;
    margin-top: 16px;
    padding: 14px 14px;
    border-radius: 12px;
    background: var(--ok-bg);
    border: 1px solid #b7e4c7;
    color: var(--ok);
    font-weight: 600;
    line-height: 1.5;
  }
  .success.show { display: block; }
  .error {
    display: none;
    margin-top: 12px;
    color: #b42318;
    font-size: 0.92rem;
  }
  .error.show { display: block; }
</style>
</head>
<body>
  <div class="card">
    <h1>模拟支付</h1>
    <p class="sub">确认订单信息后完成支付，即可返回模板商城使用模板。</p>

    <div class="field">
      <label>商品</label>
      <div class="box">` + html.EscapeString(subject) + `</div>
    </div>
    <div class="field">
      <label>业务订单</label>
      <div class="box">` + html.EscapeString(orderID) + `</div>
    </div>
    <div class="field">
      <label>支付单</label>
      <div class="box">` + html.EscapeString(payOrderID) + `</div>
    </div>
    <div class="field amount">
      <label>金额</label>
      <div class="box">¥` + strconv.FormatFloat(yuan, 'f', 2, 64) + `</div>
    </div>
    <div class="field">
      <label>状态</label>
      <div class="box"><span id="st" class="` + statusClass(paid) + `">` + statusText + `</span></div>
    </div>

    <div id="success" class="success ` + successClass + `">支付成功，点击返回即可使用模版！</div>
    <p id="err" class="error"></p>

    <div class="actions">
      <button id="btn" ` + disabledIfPaid(status) + `>确认支付</button>
      <a class="btn btn-back" id="back" href="` + html.EscapeString(mallURL) + `/">返回模板商城</a>
    </div>
  </div>
<script>
document.getElementById('btn')?.addEventListener('click', async () => {
  const btn = document.getElementById('btn');
  const err = document.getElementById('err');
  err.classList.remove('show');
  err.textContent = '';
  btn.disabled = true;
  btn.textContent = '支付处理中…';
  try {
    const res = await fetch('/api/paycallback', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        callback_id: '` + callbackID + `-' + Date.now(),
        pay_order_id: '` + payOrderID + `',
        order_id: '` + orderID + `',
        amount_fen: ` + strconv.FormatInt(amountFen, 10) + `
      })
    });
    const data = await res.json();
    if (res.ok && data.accepted) {
      const st = document.getElementById('st');
      st.textContent = '已支付';
      st.className = 'status-ok';
      document.getElementById('success').classList.add('show');
      btn.textContent = '已支付';
      return;
    }
    throw new Error(data.error || data.message || '支付未受理');
  } catch (e) {
    err.textContent = e instanceof Error ? e.message : '支付失败，请重试';
    err.classList.add('show');
    btn.disabled = false;
    btn.textContent = '确认支付';
  }
});
</script>
</body>
</html>`
}

func disabledIfPaid(status string) string {
	if status == "paid" {
		return "disabled"
	}
	return ""
}

func statusClass(paid bool) string {
	if paid {
		return "status-ok"
	}
	return ""
}
