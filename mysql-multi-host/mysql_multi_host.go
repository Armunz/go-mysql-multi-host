package mysqlmultihost

import (
	"context"
	"database/sql/driver"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

type mysqlMultiHostConnector struct {
	mu           *sync.Mutex
	connectors   []driver.Connector
	clusterDSNs  []string
	dialTimeout  time.Duration
	currentIndex int
}

func NewMySQLMultiHostConnector(clusterDSNs []string, dialTimeoutMs int) (*mysqlMultiHostConnector, error) {
	if len(clusterDSNs) == 0 {
		return nil, fmt.Errorf("multi host dsn should not be empty")
	}

	connectors := make([]driver.Connector, 0, len(clusterDSNs))
	for _, dsn := range clusterDSNs {
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mysql multi host dsn, %w", err)
		}

		connector, err := mysql.NewConnector(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create mysql connector, %w", err)
		}
		connectors = append(connectors, connector)
	}

	return &mysqlMultiHostConnector{
		mu:           &sync.Mutex{},
		connectors:   connectors,
		clusterDSNs:  clusterDSNs,
		dialTimeout:  time.Duration(dialTimeoutMs) * time.Millisecond,
		currentIndex: 0,
	}, nil
}

// Connect wraps mysql.Connect() with multiple host to handle failover.
// This function is inspired by ClickHouse handle multiple host connection.
// Ref: https://clickhouse.com/docs/integrations/go#connecting-to-multiple-nodes
func (m *mysqlMultiHostConnector) Connect(ctx context.Context) (_ driver.Conn, err error) {
	m.mu.Lock()
	start := m.currentIndex
	m.mu.Unlock()

	for i := range m.connectors {
		index := (start + i) % len(m.connectors)

		// we need to override ctx param with new context on each Connect() call
		// to prevent using expired ctx when fail to connect at current connector
		// and want to connect to next connector
		attemptCtx, cancel := context.WithTimeout(context.Background(), m.dialTimeout)
		conn, err := m.connectors[index].Connect(attemptCtx)
		if err != nil {
			log.Printf("[connect] error connecting to %s: %v\n", m.clusterDSNs[index], err)
			cancel()
			continue
		}

		// if success: update current index
		m.mu.Lock()
		m.currentIndex = index
		m.mu.Unlock()

		return &stdDriver{
			currentConn: conn,
			close:       cancel,
		}, nil
	}

	return nil, fmt.Errorf("failed to connect to any of the host provided")
}

func (m *mysqlMultiHostConnector) Driver() driver.Driver {
	return &mysql.MySQLDriver{}
}

var _ driver.Connector = (*mysqlMultiHostConnector)(nil)

type stdDriver struct {
	currentConn driver.Conn
	close       context.CancelFunc
}

var _ driver.Conn = (*stdDriver)(nil)

// Begin implements driver.Conn.
func (s *stdDriver) Begin() (driver.Tx, error) {
	return s.currentConn.Begin()
}

// Close implements driver.Conn.
func (s *stdDriver) Close() error {
	s.close()
	return s.currentConn.Close()
}

// Prepare implements driver.Conn.
func (s *stdDriver) Prepare(query string) (driver.Stmt, error) {
	return s.currentConn.Prepare(query)
}
