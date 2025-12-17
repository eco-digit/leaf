# Leaf 
## **L**ifecycle based **E**nvironmental **A**ssessment of **F**ootprints

Currently, this exporter/service is only running locally you can connect it to a prometheus server using port forwarding. 

```
leaf/
├── cmd/
│   └── leaf/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── promclient/
│   │   └── client.go
│   └── exporter/
│       └── exporter.go
├── go.mod
├── go.sum
└── config.yaml
```

```shell 
go mod init leaf  
```


```shell
go mod tidy
```

# Start exporter
```shell
go run ./cmd/leaf --config config.yaml
```
