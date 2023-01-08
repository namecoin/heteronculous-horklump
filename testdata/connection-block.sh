sudo iptables -I INPUT -p tcp --dport "$TOR_SOCKS_HOST" -i lo -j DROP
