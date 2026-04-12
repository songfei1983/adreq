default: check

check:
    gofmt -w .
    go test ./...

test:
    go test ./...

test-race:
    go test -race ./...

run:
    go run .

run-api:
    go run ./cmd/api

fmt:
    gofmt -w .

vet:
    go vet ./...

tidy:
    go mod tidy

bench:
    go test ./server -run '^$' -bench 'BenchmarkHandleRequest_' -benchmem

loadtest-direct:
    go run ./cmd/loadtest -mode=direct -duration=2s -imps=6 -cands=40 -concurrency=128

loadtest-pool:
    go run ./cmd/loadtest -mode=pool -duration=2s -imps=6 -cands=40 -concurrency=128 -pool-workers=16 -pool-queue=64k

loadtest-sweep:
    go run ./cmd/loadtest -mode=pool -sweep -duration=2s -imps=6 -cands=40 -concurrency-list=32,64,128 -pool-workers-list=16 -pool-queue-list=64,1024,64k

docker-build tag='adreq:local':
    docker build -t {{tag}} .

k8s-apply-staging:
    kubectl apply -k deploy/kustomize/overlays/staging

k8s-set-image-staging image='adreq:local':
    kubectl -n adreq-staging set image deployment/adreq adreq={{image}}

k8s-rollout-staging:
    kubectl -n adreq-staging rollout status deployment/adreq --timeout=180s

k8s-port-forward-staging port='18080':
    kubectl -n adreq-staging port-forward svc/adreq {{port}}:80

kind-load-image cluster:
    kind load docker-image adreq:local --name {{cluster}}
