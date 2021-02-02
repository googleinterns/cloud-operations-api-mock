# build stage
FROM golang as builder

ENV GO111MODULE=on

WORKDIR /server

# Copy module system over and download deps.
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy source over and build mockserver image
COPY . .
WORKDIR /server/cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mockserver

# Now build a raw image with just the mocserver
FROM scratch
COPY --from=builder /server/cmd/mockserver /server/
EXPOSE 8080
ENTRYPOINT ["/server/mockserver", "--address=:8080"]
