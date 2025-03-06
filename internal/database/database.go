package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"okxauto/internal/database/models"
)

// Database 数据库结构
type Database struct {
	db *sql.DB
}

// New 创建数据库连接
func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	// 设置数据库连接参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &Database{db: db}, nil
}

// Close 关闭数据库连接
func (db *Database) Close() error {
	return db.db.Close()
}

// 定义表结构
type TableColumn struct {
	Name     string
	Type     string
	NotNull  bool
	Default  string
	PrimaryKey bool
	Unique     bool
}

// Initialize 初始化数据库表结构
func (db *Database) Initialize() error {
	log.Println("开始初始化数据库...")

	// 定义所需的表和列
	tables := map[string][]TableColumn{
		"trades": {
			{Name: "id", Type: "INTEGER", PrimaryKey: true},
			{Name: "symbol", Type: "TEXT", NotNull: true},
			{Name: "side", Type: "TEXT", NotNull: true},
			{Name: "price", Type: "REAL", NotNull: true},
			{Name: "amount", Type: "REAL", NotNull: true},
			{Name: "strategy", Type: "TEXT", NotNull: true},
			{Name: "status", Type: "TEXT", NotNull: true},
			{Name: "order_id", Type: "TEXT", NotNull: true},
			{Name: "trade_type", Type: "TEXT", NotNull: true, Default: "'spot'"},
			{Name: "created_at", Type: "DATETIME", NotNull: true},
		},
		"users": {
			{Name: "id", Type: "INTEGER", PrimaryKey: true},
			{Name: "username", Type: "TEXT", NotNull: true, Unique: true},
			{Name: "password", Type: "TEXT", NotNull: true},
			{Name: "created_at", Type: "DATETIME", NotNull: true},
			{Name: "updated_at", Type: "DATETIME", NotNull: true},
		},
		"signals": {
			{Name: "id", Type: "INTEGER", PrimaryKey: true},
			{Name: "symbol", Type: "TEXT", NotNull: true},
			{Name: "strategy", Type: "TEXT", NotNull: true},
			{Name: "action", Type: "TEXT", NotNull: true},
			{Name: "price", Type: "REAL", NotNull: true},
			{Name: "created_at", Type: "DATETIME", NotNull: true},
		},
	}

	// 检查并创建每个表
	for tableName, columns := range tables {
		if err := db.ensureTable(tableName, columns); err != nil {
			return fmt.Errorf("确保表 %s 存在失败: %v", tableName, err)
		}
	}

	log.Println("数据库初始化完成")
	return nil
}

// ensureTable 确保表存在并且列完整
func (db *Database) ensureTable(tableName string, requiredColumns []TableColumn) error {
	// 检查表是否存在
	var exists bool
	err := db.db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("检查表是否存在失败: %v", err)
	}

	if !exists {
		// 创建表
		log.Printf("创建表 %s", tableName)
		if err := db.createTable(tableName, requiredColumns); err != nil {
			return err
		}
	} else {
		// 检查并添加缺失的列
		log.Printf("检查表 %s 的列", tableName)
		if err := db.ensureColumns(tableName, requiredColumns); err != nil {
			return err
		}
	}

	return nil
}

// createTable 创建表
func (db *Database) createTable(tableName string, columns []TableColumn) error {
	var columnDefs []string
	for _, col := range columns {
		def := fmt.Sprintf("%s %s", col.Name, col.Type)
		if col.PrimaryKey {
			def += " PRIMARY KEY AUTOINCREMENT"
		}
		if col.NotNull {
			def += " NOT NULL"
		}
		if col.Default != "" {
			def += " DEFAULT " + col.Default
		}
		if col.Unique {
			def += " UNIQUE"
		}
		columnDefs = append(columnDefs, def)
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columnDefs, ", "))
	_, err := db.db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	log.Printf("表 %s 创建成功", tableName)
	return nil
}

// ensureColumns 确保所有需要的列都存在
func (db *Database) ensureColumns(tableName string, requiredColumns []TableColumn) error {
	// 获取现有列
	rows, err := db.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return fmt.Errorf("获取表信息失败: %v", err)
	}
	defer rows.Close()

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notnull int
		var dflt_value interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk); err != nil {
			return fmt.Errorf("扫描列信息失败: %v", err)
		}
		existingColumns[strings.ToLower(name)] = true
	}

	// 添加缺失的列
	for _, col := range requiredColumns {
		if !existingColumns[strings.ToLower(col.Name)] {
			log.Printf("添加列 %s 到表 %s", col.Name, tableName)
			alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, col.Name, col.Type)
			if col.NotNull {
				if col.Default != "" {
					alterSQL += " NOT NULL DEFAULT " + col.Default
				} else {
					// SQLite不支持在ADD COLUMN时添加NOT NULL约束，除非指定DEFAULT值
					log.Printf("警告: 列 %s 将不会添加NOT NULL约束", col.Name)
				}
			}
			if col.Default != "" && !col.NotNull {
				alterSQL += " DEFAULT " + col.Default
			}
			if col.Unique {
				alterSQL += " UNIQUE"
			}
			if _, err := db.db.Exec(alterSQL); err != nil {
				return fmt.Errorf("添加列失败: %v", err)
			}
			log.Printf("列 %s 添加成功", col.Name)
		}
	}

	return nil
}

// SaveTrade 保存交易记录
func (db *Database) SaveTrade(trade *models.Trade) error {
	query := `
		INSERT INTO trades (
			symbol, side, price, amount, strategy, status, order_id, trade_type, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.db.Exec(query,
		trade.Symbol,
		trade.Side,
		trade.Price,
		trade.Amount,
		trade.Strategy,
		trade.Status,
		trade.OrderID,
		trade.TradeType,
		trade.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("保存交易记录失败: %v", err)
	}

	id, _ := result.LastInsertId()
	log.Printf("成功保存交易记录，ID=%d", id)
	return nil
}

// GetTrades 获取交易记录
func (db *Database) GetTrades(limit int) ([]*models.Trade, error) {
	query := `
		SELECT id, symbol, side, price, amount, strategy, status, order_id, trade_type, created_at
		FROM trades
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := db.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("查询交易记录失败: %v", err)
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		trade := &models.Trade{}
		err := rows.Scan(
			&trade.ID,
			&trade.Symbol,
			&trade.Side,
			&trade.Price,
			&trade.Amount,
			&trade.Strategy,
			&trade.Status,
			&trade.OrderID,
			&trade.TradeType,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描交易记录失败: %v", err)
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetTradesBySymbol 根据交易对获取交易记录
func (db *Database) GetTradesBySymbol(symbol string, limit int) ([]*models.Trade, error) {
	query := `
		SELECT id, symbol, side, price, amount, strategy, status, order_id, trade_type, created_at
		FROM trades
		WHERE symbol = ?
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := db.db.Query(query, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("查询交易记录失败: %v", err)
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		trade := &models.Trade{}
		err := rows.Scan(
			&trade.ID,
			&trade.Symbol,
			&trade.Side,
			&trade.Price,
			&trade.Amount,
			&trade.Strategy,
			&trade.Status,
			&trade.OrderID,
			&trade.TradeType,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描交易记录失败: %v", err)
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetTradesByStrategy 根据策略获取交易记录
func (db *Database) GetTradesByStrategy(strategy string, limit int) ([]*models.Trade, error) {
	query := `
		SELECT id, symbol, side, price, amount, strategy, status, order_id, trade_type, created_at
		FROM trades
		WHERE strategy = ?
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := db.db.Query(query, strategy, limit)
	if err != nil {
		return nil, fmt.Errorf("查询交易记录失败: %v", err)
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		trade := &models.Trade{}
		err := rows.Scan(
			&trade.ID,
			&trade.Symbol,
			&trade.Side,
			&trade.Price,
			&trade.Amount,
			&trade.Strategy,
			&trade.Status,
			&trade.OrderID,
			&trade.TradeType,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描交易记录失败: %v", err)
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetTradeStats 获取交易统计信息
func (db *Database) GetTradeStats(symbol string) (map[string]float64, error) {
	query := `
		SELECT 
			COUNT(*) as total_trades,
			SUM(CASE WHEN side = 'buy' THEN 1 ELSE 0 END) as buy_trades,
			SUM(CASE WHEN side = 'sell' THEN 1 ELSE 0 END) as sell_trades,
			AVG(price) as avg_price,
			SUM(amount) as total_amount
		FROM trades
		WHERE symbol = ?`

	row := db.db.QueryRow(query, symbol)

	var stats = make(map[string]float64)
	var totalTrades, buyTrades, sellTrades, avgPrice, totalAmount float64
	err := row.Scan(&totalTrades, &buyTrades, &sellTrades, &avgPrice, &totalAmount)
	if err != nil {
		return nil, fmt.Errorf("获取交易统计失败: %v", err)
	}

	stats["total_trades"] = totalTrades
	stats["buy_trades"] = buyTrades
	stats["sell_trades"] = sellTrades
	stats["avg_price"] = avgPrice
	stats["total_amount"] = totalAmount

	return stats, nil
}

// GetTradeHistory 获取交易历史记录
func (db *Database) GetTradeHistory(limit int) ([]*models.Trade, error) {
	// 这里直接调用已有的 GetTrades 方法，因为功能是一样的
	return db.GetTrades(limit)
}

// 如果需要更详细的交易历史记录，可以使用这个版本：
/*
func (db *Database) GetTradeHistory(params map[string]interface{}) ([]*models.Trade, error) {
	query := `
		SELECT id, symbol, side, price, amount, strategy, status, order_id, trade_type, created_at
		FROM trades
		WHERE 1=1`
	
	args := make([]interface{}, 0)
	
	// 添加过滤条件
	if symbol, ok := params["symbol"].(string); ok && symbol != "" {
		query += " AND symbol = ?"
		args = append(args, symbol)
	}
	
	if strategy, ok := params["strategy"].(string); ok && strategy != "" {
		query += " AND strategy = ?"
		args = append(args, strategy)
	}
	
	if tradeType, ok := params["trade_type"].(string); ok && tradeType != "" {
		query += " AND trade_type = ?"
		args = append(args, tradeType)
	}
	
	// 添加时间范围
	if startTime, ok := params["start_time"].(time.Time); ok {
		query += " AND created_at >= ?"
		args = append(args, startTime)
	}
	
	if endTime, ok := params["end_time"].(time.Time); ok {
		query += " AND created_at <= ?"
		args = append(args, endTime)
	}
	
	// 添加排序和限制
	query += " ORDER BY created_at DESC"
	if limit, ok := params["limit"].(int); ok && limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询交易历史失败: %v", err)
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		trade := &models.Trade{}
		err := rows.Scan(
			&trade.ID,
			&trade.Symbol,
			&trade.Side,
			&trade.Price,
			&trade.Amount,
			&trade.Strategy,
			&trade.Status,
			&trade.OrderID,
			&trade.TradeType,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描交易记录失败: %v", err)
		}
		trades = append(trades, trade)
	}

	return trades, nil
}
*/ 