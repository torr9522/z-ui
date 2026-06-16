# X-UI

[简体中文](./README.md)| ENGLISH  
X-UI is a webUI panel based on Xray-core which supports multi protocols and multi users  
This project is a fork of [vaxilu&#39;s project](https://github.com/vaxilu/x-ui),and it is a experiental project which used by myself for learning golang   
If you need more language options ,please open a issue and let me know that

# Changes   
- 2023.07.18：Random Reality dest and serverNames;more detailed sniffing settings available  
- 2023.06.10：Enable TLS will reuse panel's certs and domain;add setting for ocspStapling;refactor device limit  
- 2023.04.09：Support REALITY for now  
- 2023.03.05：User expiry time limit for each user  
- 2023.02.09：User traffic limit for each user,support utls sharing link  
- 2022.12.07：Add device limit and more tls configuration  
- 2022.11.15：Add xtls-rprx-vision flow option;cron job for geo update and log clear    
- 2022.10.23：Fully support for English,add export links,add CPU cores display
- 2022.08.11：Support multi users on the same port;add CPU limit exceed  alert  
- 2022.07.28：Add acme standalone mode for cert issue；add  mechanism to keep X-UI alive even there exist crashes
- 2022.07.24：Add base path auto generate feature for security;add traffice reset automatically;add device alert
- 2022.07.21：Add more translations;add restart/stop xray service in Web panel
- 2022.07.11：Add time expiration notify for each inbound;add traffic limit notify for each inbound;add get url link command/inbound copy command in telegram bot  
- 2022.07.03：Add transport options in Trojan protocol;restruct Telegram bot for convenience  
- 2022.06.19：Add shadowsocks 2022 Ciphers,add inbounds search,traffic clear function in WebUI
- 2022.05.14：Add Telegram bot commands,support enable/disable/delete/status check
- 2022.04.25：Add SSH login notify
- 2022.04.23：Add WebUi login notify
- 2022.04.16：Add Telegram bot set up in WebUi pannel
- 2022.04.12：Optimize Telegram bot notify,more human friendly
- 2022.04.06：Add cert issue function，optimize installation/update and add telegram bot notify

# Basics

- support system status info check
- support multi protocols and multi users
- support protocols：vmess、vless、trojan、shadowsocks、dokodemo-door、socks、http
- support many transport method including tcp、udp、ws、kcp etc
- traffic counting,traffic restrict and time restrcit
- support custom configuration template
- support https access fot WebUI
- support SSL cert issue by Acme
- support telegram bot notify and control
- more functions in control menu  

for more detailed usages,plz see [WIKI](https://github.com/FranzKafkaYu/x-ui/wiki)

# Installation
Make sure `bash`, `curl`, `systemd`, and network access are available. The installer supports Debian, Ubuntu, CentOS, Rocky, and Alma on amd64/arm64.

The installer generates a random username, random password, and a random panel port in the 10000-59999 range. The login credentials are printed only to the current terminal.

```
bash <(curl -Ls https://raw.githubusercontent.com/FranzKafkaYu/x-ui/master/install.sh)
```  
To install a specific version:
```
bash <(curl -Ls https://raw.githubusercontent.com/FranzKafkaYu/x-ui/master/install.sh) beta-v0.1.0
``` 

## Shortcut  
After installation, use the `x-ui` command:
```
x-ui start
x-ui stop
x-ui restart
x-ui status
x-ui enable
x-ui disable
x-ui log
x-ui reset-user
x-ui reset-port
x-ui info
x-ui cert
x-ui cert status
x-ui cert renew
x-ui cert autorenew
x-ui update
x-ui uninstall
```

# System requirements:  
## MEM  
- 128MB minimal/256MB+ recommend  
## OS
- CentOS 7+
- Ubuntu 16+
- Debian 8+

# Telegram

[Channel](https://t.me/CoderfanBaby)  
[Group](https://t.me/franzkafayu)

# Credits
- [vaxilu/x-ui](https://github.com/vaxilu/x-ui)
- [XTLS/Xray-core](https://github.com/XTLS/Xray-core)
- [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)  

# Sponsor  

if you want to purchase some virtual servers,you can purchase by my aff link:   
- [BandwagonHost](https://bandwagonhost.com/aff.php?aff=65703)     
- [Cloudcone](https://app.cloudcone.com/?ref=7536)  
- [SpartanHost](https://billing.spartanhost.net/aff.php?aff=1875)  


## Stargazers over time

[![Stargazers over time](https://starchart.cc/FranzKafkaYu/x-ui.svg)](https://starchart.cc/FranzKafkaYu/x-ui)
