.PHONY: protoc/go
protoc/go:
	docker run -v $(CURDIR):/defs -w /defs namely/protoc-all -d ./api/v0 -l go --go-module-prefix github.com/ipni/depute -o .
