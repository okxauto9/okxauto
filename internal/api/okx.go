package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
	
	"okxauto/internal/models"
	"okxauto/internal/utils"
)

type OKXClient struct {
	apiKey      string
	secretKey   string
	passphrase  string
	baseURL     string
	client      *http.Client
	isSimulated bool // 新增字段，标记是否是模拟盘
	lastRequest time.Time    // 添加上次请求时间
	mu          sync.Mutex   // 添加互斥锁以保证并发安全
}

func NewOKXClient(apiKey, secretKey, passphrase string, mode string) *OKXClient {
	// 根据模式选择不同的API地址
	baseURL := "https://www.okx.com"
	isSimulated := false

	if mode == "simulation" {
		isSimulated = true
		// 模拟盘API地址保持不变，但需要添加特殊标记
		log.Printf("使用模拟盘模式")
	} else {
		log.Printf("使用实盘模式")
	}

	client := &OKXClient{
		apiKey:     apiKey,
		secretKey:  secretKey,
		passphrase: passphrase,
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 10 * time.Second},
		isSimulated: isSimulated,
		lastRequest: time.Now(),  // 初始化上次请求时间
	}

	return client
}

// 生成签名
func (c *OKXClient) sign(timestamp, method, requestPath string, body []byte) string {
	message := timestamp + method + requestPath
	if len(body) > 0 {
		message += string(body)
	}

	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// 发送请求
func (c *OKXClient) sendRequest(method, path string, body interface{}) ([]byte, error) {
	return c.sendRequestWithRetry(method, path, body, utils.DefaultRetryConfig)
}

func (c *OKXClient) sendRequestWithRetry(method, path string, body interface{}, retryConfig utils.RetryConfig) ([]byte, error) {
	var resp []byte
	err := utils.RetryOperation(func() error {
		var sendErr error
		resp, sendErr = c.doRequest(method, path, body)
		return sendErr
	}, retryConfig)
	
	return resp, err
}

// doRequest 执行实际的HTTP请求
func (c *OKXClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	c.mu.Lock()
	now := time.Now()
	if diff := now.Sub(c.lastRequest); diff < time.Second/6 {  // 限制为6次/秒
		time.Sleep(time.Second/6 - diff)
	}
	c.lastRequest = now
	c.mu.Unlock()

	var bodyJSON []byte
	var err error

	if body != nil {
		bodyJSON, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	sign := c.sign(timestamp, method, path, bodyJSON)

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OK-ACCESS-KEY", c.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", sign)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", c.passphrase)

	if c.isSimulated {
		req.Header.Set("x-simulated-trading", "1")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查API响应
	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Code != "0" {
		return nil, fmt.Errorf("API错误: %s", string(respBody))
	}

	return respBody, nil
}

// 修改下单请求结构
type PlaceOrderRequest struct {
	InstId  string    `json:"instId"`          // 产品ID
	TdMode  string    `json:"tdMode"`          // 交易模式：cash/cross/isolated
	Side    OrderSide `json:"side"`            // 订单方向
	PosSide string    `json:"posSide"`         // 持仓方向：long/short
	OrdType OrderType `json:"ordType"`         // 订单类型：market/limit
	Sz      string    `json:"sz"`              // 委托数量
	Px      string    `json:"px,omitempty"`    // 委托价格，市价单不需要
	Lever   string    `json:"lever,omitempty"` // 杠杆倍数
	ClOrdId string    `json:"clOrdId,omitempty"` // 客户自定义订单ID
}

// 修改下单方法
func (c *OKXClient) PlaceOrder(req *PlaceOrderRequest) (*OrderResponse, error) {
	// 生成一个简单的数字ID
	if req.ClOrdId == "" {
		// 使用Unix时间戳的后12位作为ID
		now := time.Now().UnixNano()
		req.ClOrdId = fmt.Sprintf("%012d", now%1000000000000)
	}

	// 确保交易模式正确
	if req.TdMode == "" {
		req.TdMode = "isolated"
	}

	// 修改合约张数格式
	if sz, err := strconv.ParseFloat(req.Sz, 64); err == nil {
		req.Sz = fmt.Sprintf("%.0f", sz) // 确保是整数
	}

	// 打印完整请求内容用于调试
	reqJSON, _ := json.MarshalIndent(req, "", "  ")
	log.Printf("发送下单请求: %s", string(reqJSON))

	resp, err := c.sendRequest("POST", "/api/v5/trade/order", req)
	if err != nil {
		return nil, err
	}

	// 打印原始响应用于调试
	log.Printf("收到下单响应: %s", string(resp))

	var result struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data []OrderResponse `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(resp))
	}

	if result.Code != "0" {
		if len(result.Data) > 0 && result.Data[0].SCode != "" {
			return nil, fmt.Errorf("下单失败: %s (错误码: %s)", result.Data[0].SMsg, result.Data[0].SCode)
		}
		return nil, fmt.Errorf("下单失败: %s (code=%s)", result.Msg, result.Code)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("下单响应数据为空")
	}

	return &result.Data[0], nil
}

// Balance 结构体定义
type Balance struct {
	Currency  string `json:"ccy"`      // 币种，如 BTC
	Balance   string `json:"bal"`      // 余额
	Available string `json:"availBal"` // 可用余额
	Frozen    string `json:"frozenBal"`// 冻结余额
}

// GetBalances 获取账户所有货币余额
func (c *OKXClient) GetBalances() ([]*Balance, error) {
	// 使用正确的API路径获取账户余额
	resp, err := c.sendRequest("GET", "/api/v5/account/balance", nil)
	if err != nil {
		return nil, err
	}

	// 打印原始响应用于调试
	log.Printf("获取余额响应: %s", string(resp))

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			AdjEq       string `json:"adjEq"`       // 调整后权益
			Details     []struct {
				AvailBal    string `json:"availBal"`    // 可用余额
				AvailEq     string `json:"availEq"`     // 可用权益
				CashBal     string `json:"cashBal"`     // 现金余额
				Ccy         string `json:"ccy"`         // 币种
				CrossLiab   string `json:"crossLiab"`   // 全仓负债
				DisEq       string `json:"disEq"`       // 美金层面币种折算权益
				Eq          string `json:"eq"`          // 币种总权益
				FrozenBal   string `json:"frozenBal"`   // 冻结余额
				Interest    string `json:"interest"`    // 计息
				IsoEq       string `json:"isoEq"`       // 逐仓权益
				IsoLiab     string `json:"isoLiab"`     // 逐仓负债
				IsoUpl      string `json:"isoUpl"`      // 逐仓未实现盈亏
				Liab        string `json:"liab"`        // 负债
				MaxLoan     string `json:"maxLoan"`     // 最大可借
				MgnRatio    string `json:"mgnRatio"`    // 保证金率
				NotionalLever string `json:"notionalLever"` // 杠杆倍数
				OrdFrozen   string `json:"ordFrozen"`   // 委托冻结数量
				Twap        string `json:"twap"`        // 当前负债币种触发系统自动换币的风险
				Upl         string `json:"upl"`         // 未实现盈亏
				UplLiab     string `json:"uplLiab"`     // 未实现负债
				StgyEq      string `json:"stgyEq"`      // 策略权益
			} `json:"details"`
			Imr         string `json:"imr"`         // 初始保证金
			IsoEq       string `json:"isoEq"`       // 逐仓仓位权益
			MgnRatio    string `json:"mgnRatio"`    // 保证金率
			Mmr         string `json:"mmr"`         // 维持保证金
			NotionalUsd string `json:"notionalUsd"` // 以美金价值为单位的持仓数量
			OrdFroz     string `json:"ordFroz"`     // 订单冻结保证金
			TotalEq     string `json:"totalEq"`     // 美金层面权益
			UplLiab     string `json:"uplLiab"`     // 未实现亏损
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析余额响应失败: %v, 响应内容: %s", err, string(resp))
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("获取余额失败: %s", result.Msg)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("账户余额为空")
	}

	// 打印账户总权益
	if len(result.Data) > 0 {
		log.Printf("账户总权益: %s USDT", result.Data[0].TotalEq)
	}

	// 只返回有余额的币种
	balances := make([]*Balance, 0)
	for _, data := range result.Data {
		for _, detail := range data.Details {
			// 使用eq(总权益)来判断是否有余额
			eq, _ := strconv.ParseFloat(detail.Eq, 64)
			if eq > 0 {
				balances = append(balances, &Balance{
					Currency:  detail.Ccy,
					Balance:   detail.Eq,         // 使用总权益作为余额
					Available: detail.AvailEq,    // 使用可用权益作为可用余额
					Frozen:    detail.FrozenBal,  // 冻结余额
				})
				// 打印详细的余额信息
				log.Printf("币种详情 - %s: 总权益=%s, 可用=%s, 冻结=%s, 现金=%s", 
					detail.Ccy, detail.Eq, detail.AvailEq, detail.FrozenBal, detail.CashBal)
			}
		}
	}

	return balances, nil
}

// GetBalance 获取指定货币余额
func (c *OKXClient) GetBalance(currency string) (*Balance, error) {
	balances, err := c.GetBalances()
	if err != nil {
		return nil, err
	}

	for _, balance := range balances {
		if balance.Currency == currency {
			return balance, nil
		}
	}

	return nil, fmt.Errorf("未找到货币: %s", currency)
}

// 获取K线数据
func (c *OKXClient) GetKlines(symbol string, period string, limit int) ([]Candle, error) {
	path := fmt.Sprintf("/api/v5/market/candles?instId=%s&bar=%s&limit=%d", symbol, period, limit)

	resp, err := c.sendRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code string     `json:"code"`
		Data [][]string `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	candles := make([]Candle, 0, len(result.Data))
	for _, item := range result.Data {
		if len(item) >= 6 {
			candle := Candle{
				Timestamp: item[0],
				Open:      item[1],
				High:      item[2],
				Low:       item[3],
				Close:     item[4],
				Volume:    item[5],
			}
			candles = append(candles, candle)
		}
	}

	return candles, nil
}

// 取消订单
func (c *OKXClient) CancelOrder(symbol, orderId string) error {
	req := struct {
		InstId  string `json:"instId"`
		OrderId string `json:"ordId"`
	}{
		InstId:  symbol,
		OrderId: orderId,
	}

	resp, err := c.sendRequest("POST", "/api/v5/trade/cancel-order", req)
	if err != nil {
		return err
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	if result.Code != "0" {
		return fmt.Errorf("取消订单失败: %s", result.Msg)
	}

	return nil
}

// SetLeverage 设置杠杆倍数
func (c *OKXClient) SetLeverage(instId string, lever string, mgnMode string, posSide string) error {
	req := struct {
		InstId  string `json:"instId"`
		Lever   string `json:"lever"`
		MgnMode string `json:"mgnMode"`
		PosSide string `json:"posSide,omitempty"`
	}{
		InstId:  instId,
		Lever:   lever,
		MgnMode: mgnMode,
		PosSide: posSide,
	}

	resp, err := c.sendRequest("POST", "/api/v5/account/set-leverage", req)
	if err != nil {
		return err
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != "0" {
		return fmt.Errorf("设置杠杆倍数失败: %s", result.Msg)
	}

	return nil
}

// GetPositions 方法返回 models.Position
func (c *OKXClient) GetPositions(instId string) ([]*models.Position, error) {
	path := fmt.Sprintf("/api/v5/account/positions?instId=%s", instId)
	resp, err := c.sendRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstId    string `json:"instId"`    
			PosSide   string `json:"posSide"`   
			Pos       string `json:"pos"`       
			AvgPx     string `json:"avgPx"`     
			UPL       string `json:"upl"`       
			UplRatio  string `json:"uplRatio"`  
			Lever     string `json:"lever"`     
			MgnMode   string `json:"mgnMode"`   
			MgnRatio  string `json:"mgnRatio"`  // 保证金率
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析持仓信息失败: %v", err)
	}

	positions := make([]*models.Position, 0)
	for _, pos := range result.Data {
		position, _ := strconv.ParseFloat(pos.Pos, 64)
		avgPrice, _ := strconv.ParseFloat(pos.AvgPx, 64)
		unPnL, _ := strconv.ParseFloat(pos.UPL, 64)
		pnlRatio, _ := strconv.ParseFloat(pos.UplRatio, 64)
		marginRatio, _ := strconv.ParseFloat(pos.MgnRatio, 64)

		if position != 0 {
			positions = append(positions, &models.Position{
				Symbol:      pos.InstId,
				PosSide:    pos.PosSide,
				Position:   position,
				AvgPrice:   avgPrice,
				UnrealPnL:  unPnL,
				PnLRatio:   pnlRatio,
				MarginRatio: marginRatio,
			})
			
			log.Printf("持仓信息 - 交易对: %s, 方向: %s, 数量: %.2f, 均价: %.4f, 收益率: %.2f%%, 保证金率: %.2f%%, 未实现盈亏: %.2f USDT",
				pos.InstId, pos.PosSide, position, avgPrice, pnlRatio*100, marginRatio*100, unPnL)
		}
	}

	return positions, nil
}

// AddMargin 追加或减少保证金
func (c *OKXClient) AddMargin(params map[string]string) (map[string]interface{}, error) {
	resp, err := c.sendRequest("POST", "/api/v5/account/position/margin-balance", params)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return result, nil
}
