# Build with the golang image
FROM golang:1.26-alpine AS build

# Add git
RUN apk add --no-cache git

# Set workdir
WORKDIR /work

# Add dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 go build

# Generate final image
FROM scratch
USER 1000
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /work/bol /bol
ENTRYPOINT [ "/bol" ]
