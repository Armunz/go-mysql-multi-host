# go-mysql-multi-host
Golang MySQL Multi Host Connector. 

This package wraps mysql.Connect() with multiple host connector to handle failover when the current host is down.

# Installation

```
go get github.com/Armunz/go-mysql-multi-host
```

# Usage

```
import (
    "database/sql"

    "github.com/Armunz/go-mysql-multi-host"
)

hosts := []string{
    "<user>:<password>@tcp(<host-1>:<port-1>)/<database>",
    "<user>:<password>@tcp(<host-2>:<port-2>)/<database>",
    "<user>:<password>@tcp(<host-3>:<port-3>)/<database>",
}

dialTimeoutMs := 3000

mysqlMultiHostConnector := mysqlmultihost.NewMySQLMultiHostConnector(hosts, dialTimeoutMs)
db := sql.OpenDB(mysqlMultiHostConnector)
```
