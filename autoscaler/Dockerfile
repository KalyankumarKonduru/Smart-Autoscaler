# ---- build stage ----
    FROM golang:1.22 AS build
    WORKDIR /src
    
    # download deps (creates an initial go.sum)
    COPY go.mod ./
    RUN go mod download
    
    # copy sources, then reconcile imports -> go.sum
    COPY . .
    RUN --mount=type=cache,target=/go/pkg/mod \
        --mount=type=cache,target=/root/.cache/go-build \
        go mod tidy
    
    # build
    RUN --mount=type=cache,target=/go/pkg/mod \
        --mount=type=cache,target=/root/.cache/go-build \
        CGO_ENABLED=0 go build -o /out/autoscaler ./cmd/autoscaler
    
    # ---- runtime ----
    FROM gcr.io/distroless/base-debian11
    COPY --from=build /out/autoscaler /autoscaler
    ENV PORT=8080
    EXPOSE 8080
    ENTRYPOINT ["/autoscaler"]
    