@echo off

set PROXY_SERVER=




rem Show current IP address (without proxy)
echo Current IP address (without proxy):
curl myip.ipip.net


rem Set temporary environment variables to check proxy
set ALL_PROXY=
set all_proxy=
set   https_proxy=
set   http_proxy=
set   all_proxy=

rem Check if proxy is working
echo IP address (with proxy):
curl myip.ipip.net

echo Proxy has been set to %PROXY_SERVER%
pause