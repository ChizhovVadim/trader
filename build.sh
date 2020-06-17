gitRevision=$(git rev-list -1 HEAD)
buildDate=$(date +"%Y-%m-%d")
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.buildDate=$buildDate -X main.gitRevision=$gitRevision" -o luatrader.exe luatrader
