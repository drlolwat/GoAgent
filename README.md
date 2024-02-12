# GoAgent

Build (Linux):
```
go build -o goagent -ldflags "-s -w -X main.CLIENT_UUID=<YOUR_CLIENT_UUID> -X main.CLIENT_KEY=<YOUR_CLIENT_KEY>" .
```

Build (Windows):
```
GOOS=windows GOARCH=amd64 go build -o goagent.exe -ldflags "-s -w -X main.CLIENT_UUID=<YOUR_CLIENT_UUID> -X main.CLIENT_KEY=<YOUR_CLIENT_KEY>" .
```