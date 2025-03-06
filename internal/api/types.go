/*
 * @Author: okxauto9@gmail.com
 * @Date: 2025-02-04 22:39:58
 * @LastEditors: okxauto9@gmail.com
 * @LastEditTime: 2025-02-05 13:37:35
 * @FilePath: \okx-bot3\internal\api\types.go
 * @Description:
 *
 * Copyright (c) 2025 by okxauto9@gmail.com, All Rights Reserved.
 */
package api

// OrderSide 订单方向
type OrderSide string

const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

// OrderType 订单类型
type OrderType string

const (
	Market OrderType = "market"
	Limit  OrderType = "limit"
)

// OrderResponse 下单响应
type OrderResponse struct {
	OrderId string `json:"ordId"`
	ClOrdId string `json:"clOrdId"`
	Tag     string `json:"tag"`
	SCode   string `json:"sCode"`
	SMsg    string `json:"sMsg"`
}

// Candle K线数据
type Candle struct {
	Timestamp string `json:"ts"`
	Open      string `json:"o"`
	High      string `json:"h"`
	Low       string `json:"l"`
	Close     string `json:"c"`
	Volume    string `json:"vol"`
}
