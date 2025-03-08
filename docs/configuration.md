# OKX自动交易系统配置说明

一个基于OKX交易所API的自动化交易系统，支持合约交易，提供多种交易策略和风险管理功能。
- 注意：使用本程序先使用模拟盘练习
- 注意：投资有风险，入市需谨慎。提高警惕，小心上当受骗。投资切记不要影响原有生活质量，更不要借贷投资！！！

## 系统要求

- Go 1.21+
- SQLite3
- GCC (用于CGO编译)
- Windows/Linux/MacOS

## 快速开始

### 1. 配置文件设置

在 `config/config.yaml` 中配置您的API信息和交易参数：

```yaml
api:
  key: "your-api-key"
  secret: "your-api-secret"
  passphrase: "your-passphrase"
  mode: "simulation"  # simulation模拟盘或live实盘

trading:
  mode: "simulation"    
  trade_type: "futures"  
  leverage: 5       
  margin_mode: "isolated" 
  reserve_balance: 200.11  # USDT预留余额
```

### 2. 编译

```bash
# Linux/MacOS
./build.sh

# Windows
# 确保安装了MinGW-w64
./build.sh
```

### 3. 运行

```bash
./okxauto -config config/config.yaml
```

## 配置说明

### API配置

```yaml
api:
  key: "xxxxxxxxxxxxxxxxx"
  secret: "xxxxxxxxxxxxxxxxx"
  passphrase: "xxxxxxxxxxxxxxxxx"
  mode: "simulation"  # simulation或live
  base_url: "https://www.okx.com"
```

### 交易配置

```yaml
trading:
  mode: "simulation"    
  trade_type: "futures"  
  leverage: 5       
  margin_mode: "isolated" 
  
  # 做多配置
  long_position:
    enabled: true
    entry_range:
      min: 1.2666  
      max: 1.3666  
    take_profit: 0.5  # 止盈率
    stop_loss: 0.3    # 止损率
    
  # 做空配置
  short_position:
    enabled: true
    entry_range:
      min: 1.9666 
      max: 2.2666  
    take_profit: 0.5
    stop_loss: 0.3
```

## API接口

### 认证接口
- POST /api/login - 用户登录

### 交易接口
- GET /api/trades/history - 获取交易历史
- GET /api/trades/active - 获取活跃交易

### 策略接口
- GET /api/strategies - 获取策略列表
- POST /api/strategies/:name/enable - 启用策略
- POST /api/strategies/:name/disable - 禁用策略
- PUT /api/strategies/:name/config - 更新策略配置

### 系统接口
- GET /api/system/status - 获取系统状态
- GET /api/system/balance - 获取账户余额


## 文档

有关 okxauto9 的更多详细信息，请参阅此处的文档文件：: [docs/](docs/)


## 支持

- 对程序有任何想法和建议可以联系，或有定制版本需求；
[![Twitter](https://img.shields.io/badge/Twitter-@okxauto9-1DA1F2?logo=twitter)](https://x.com/okxauto9)
[![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)](https://t.me/okxauto9)


## 捐赠：
如果你认为本项目程序有价值，请考虑捐赠以表达对其发展的感激之情：
**推荐链接**  
欢迎使用这些推荐进行注册：
**[欧易](https://www.okx.com/join/63236562)


## 许可证
版权所有 © 2025 okxauto9@gmail.com 