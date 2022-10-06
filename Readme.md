## Demo Configuration Module

Sequence of configuration loading
- default configuration
- config.json
- environ vars
- flags

```bash
# default config from config.json
go run cmd/run-once/main.go 

# Override from environ
BUILD_NUMBER=8.0.0 go run cmd/run-once/main.go 

# Override from flag/cmd line arg
BUILD_NUMBER=8.0.0 go run cmd/run-once/main.go -build_number 7.0.0
```
