@echo off

set PROXY_SERVER=socks5://127.0.0.1:33211

rem Set environment variables
setx ALL_PROXY %PROXY_SERVER%
setx all_proxy %PROXY_SERVER%

rem Show current IP address (without proxy)
echo Current IP address (without proxy):
curl myip.ipip.net


rem Set temporary environment variables to check proxy
set ALL_PROXY=%PROXY_SERVER%
set all_proxy=%PROXY_SERVER%
set   https_proxy=%PROXY_SERVER%
set   http_proxy=%PROXY_SERVER%
set   all_proxy=%PROXY_SERVER%

rem Check if proxy is working
echo IP address (with proxy):
curl myip.ipip.net

echo Proxy has been set to %PROXY_SERVER%
pause