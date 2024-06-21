# gam

[![Go Reference](https://pkg.go.dev/badge/github.com/Makepad-fr/gam.svg)](https://pkg.go.dev/github.com/Makepad-fr/gam)
[![Go Report Card](https://goreportcard.com/badge/github.com/Makepad-fr/gam)](https://goreportcard.com/report/github.com/Makepad-fr/gam)

An analytics middleware for Go web servers

## Install


**TIP:** If you want to use a specific version please refer [releases](https://github.com/Makepad-fr/gam/releases) and replace `latest` with the desired version.

```bash
go get -u https://github.com/Makepad-fr/gam@latest
```

## Usage

First create a new instance of gam by using the [`gam.Init` function](https://pkg.go.dev/github.com/Makepad-fr/gam#Init)

```go
g, err := gam.Init("ch_username", "ch_password", "ch_hostname", "ch_port", "ch_database_name", "ch_table_name", true, true)
```

This will return an error if there's a connection issue or if the table names `ch_table_name` exists but have a different schema then the [expected schema](https://github.com/Makepad-fr/gam/blob/v1.0.0-rc5/db.go#L11-L45). Otherwise it will create an table named `ch_table_name` if it does not exists yet.

Then pass your handler function to the `g.Middleware` as you'd done with any middleware functions in go

```go
h := g.Middleware(myHandlerFunc)
```

It will create you a `http.Handler` that you can pass to a `http.ServerMux` or `http.Server` using `Handle` method



