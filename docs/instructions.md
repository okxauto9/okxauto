# 使用说明

下面是使用本程序的概述。

- 注意：使用本程序先使用模拟盘练习
- 注意：投资有风险，入市需谨慎。提高警惕，小心上当受骗。投资切记不要影响原有生活质量，更不要借贷投资！！！

##  创建我的APIKey

- 点击跳转至官网创建V5APIKey的页面 创建我的APIKey（https://www.okx.com/zh-hans/account/my-api）
- 生成APIKey

- 在对任何请求进行签名之前，您必须通过交易网站创建一个APIKey。创建APIKey后，您将获得3个必须记住的信息：

  - APIKey
  - SecretKey
  - Passphrase

- APIKey和SecretKey将由平台随机生成和提供，Passphrase将由您提供以确保API访问的安全性。平台将存储Passphrase加密后的哈希值进行验证，但如果您忘记Passphrase，则无法恢复，请您通过交易网站重新生成新的APIKey。

- API key 有如下3种权限，一个 API key 可以有一个或多个权限。

  - 读取 ：查询账单和历史记录等 读权限
  - 提现 ：可以进行提币
  - 交易 ：可以下单和撤单，转账，调整配置 等写权限

- 每个API key最多可绑定20个IP地址
- 未绑定IP且拥有交易或提币权限的APIKey，将在闲置14天之后自动删除。(模拟盘的 API key 不会被删除) 

##  下载程序

- 根据你的系统下载本程序
- win下载https://github.com/okxauto9/okxauto/releases/download/v1.0/okxauto-Windows_x86_64.v1.0.zip
- linux下载https://github.com/okxauto9/okxauto/releases/download/v1.0/okxauto-Linux_x86_64.v1.0.zip

- 配置文件设置

- 在 `config/config.yaml` 中配置您的API信息和交易参数：

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
##  启动程序

- win使用./Win-Okxauto_start.bat
- win使用./Win-Proxy_okxauto_start.bat （国内需使用代理，非国内IP才能连接OKX API）
- linux使用./okxauto-Linux_x86_64.v1.0


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
