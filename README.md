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

mysqlMultiHostConnector := mysqlmultihost.NewMySQLMultiHostConnector(hosts, dialTimeoutMs)
db := sql.OpenDB(mysqlMultiHostConnector)
```
