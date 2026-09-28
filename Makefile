# Local quality gates for ghs (constitution Principle V).
#
#   make fmt    gofmt -l must print nothing
#   make vet    go vet ./...
#   make test   full suite with the race detector (unit + end-to-end)
#   make e2e    only the hermetic end-to-end suite in internal/e2e
#   make check  fmt, vet, test (what CI runs)
#
# The end-to-end suite runs the real ghs binary against fake gh, git, ssh, and
# ssh-keygen executables in a throwaway HOME; it needs no network and never
# touches your real ~/.ssh, ~/.gitconfig, or GitHub CLI state.

.NOTPARALLEL:
.PHONY: fmt vet test e2e check

fmt:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

e2e:
	go test -race -count=1 ./internal/e2e/...

check: fmt vet test
