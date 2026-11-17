FROM golang:1.23 AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# CGO disabled + static build: runs clean in a scratch/distroless final image
RUN CGO_ENABLED=0 GOOS=linux go build -o /triage-agent .

FROM gcr.io/distroless/static-debian12
COPY --from=build /triage-agent /triage-agent
USER 65532:65532
ENTRYPOINT ["/triage-agent"]
