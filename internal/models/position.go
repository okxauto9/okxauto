/*
 * @Author: okxauto9@gmail.com
 * @Date: 2025-02-13 18:54:41
 * @LastEditors: okxauto9@gmail.com
 * @LastEditTime: 2025-02-13 19:01:40
 * @FilePath: \ok-bot1\internal\models\position.go
 * @Description:
 *
 * Copyright (c) 2025 by okxauto9@gmail.com, All Rights Reserved.
 */
package models

// Position 持仓信息
type Position struct {
	Symbol    string  `json:"symbol"`
	PosSide   string  `json:"posSide"`  // long/short
	Position  float64 `json:"pos"`      // 持仓数量
	AvgPrice  float64 `json:"avgPx"`    // 开仓均价
	UnrealPnL float64 `json:"upl"`      // 未实现盈亏
	PnLRatio  float64 `json:"uplRatio"` // 收益率
	MarginRatio float64 `json:"mgnRatio"` // 保证金率
}
