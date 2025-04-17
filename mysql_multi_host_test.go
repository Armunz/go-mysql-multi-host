package mysqlmultihost

import (
	"context"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockConnector is a mock implementation of driver.Connector.
type MockConnector struct {
	shouldFail bool
}

func (m *MockConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if m.shouldFail {
		return nil, errors.New("connection failed")
	}

	return &MockConn{}, nil
}

func (m *MockConnector) Driver() driver.Driver {
	return nil
}

type MockConn struct{}

func (m *MockConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (m *MockConn) Close() error                              { return nil }
func (m *MockConn) Begin() (driver.Tx, error)                 { return nil, nil }

func TestNewMySQLMultiHostConnector(t *testing.T) {
	type args struct {
		clusterDSNs   []string
		dialTimeoutMs int
	}
	tests := []struct {
		name    string
		args    args
		want    *mysqlMultiHostConnector
		wantErr bool
	}{
		{
			name: "When all is good, then it will return mysql multi host connector object",
			args: args{
				clusterDSNs: []string{
					"user:password@tcp(localhost:3306)/some_db",
					"user:password@tcp(localhost:3307)/some_db",
					"user:password@tcp(localhost:3308)/some_db",
				},
				dialTimeoutMs: 3000,
			},
		},
		{
			name: "When clusterDSNs is empty, then it will return error",
			args: args{
				clusterDSNs:   []string{},
				dialTimeoutMs: 3000,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "When dialTimeoutMs is less than 0, then it will return error",
			args: args{
				clusterDSNs: []string{
					"user:password@tcp(localhost:3306)/some_db",
					"user:password@tcp(localhost:3307)/some_db",
					"user:password@tcp(localhost:3308)/some_db",
				},
				dialTimeoutMs: 0,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "When failed to parse DSN, then it will return error",
			args: args{
				clusterDSNs: []string{
					"invalid",
					"user:password@tcp(localhost:3307)/some_db",
					"user:password@tcp(localhost:3308)/some_db",
				},
				dialTimeoutMs: 3000,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMySQLMultiHostConnector(tt.args.clusterDSNs, tt.args.dialTimeoutMs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMySQLMultiHostConnector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_mysqlMultiHostConnector_Connect(t *testing.T) {
	ctx := context.Background()

	t.Run("When first connector can connect, then return the connector with no error", func(t *testing.T) {
		mockConnectors := []driver.Connector{
			&MockConnector{shouldFail: false},
			&MockConnector{shouldFail: false},
		}

		connector := &mysqlMultiHostConnector{
			mu:           &sync.Mutex{},
			connectors:   mockConnectors,
			clusterDSNs:  []string{"host1", "host2"},
			dialTimeout:  3000 * time.Millisecond,
			currentIndex: 0,
		}

		mockDriver, err := connector.Connect(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, mockDriver)
	})

	t.Run("When first connector can't connect, then continue to next connector", func(t *testing.T) {
		mockConnectors := []driver.Connector{
			&MockConnector{shouldFail: true},
			&MockConnector{shouldFail: false},
		}

		connector := &mysqlMultiHostConnector{
			mu:           &sync.Mutex{},
			connectors:   mockConnectors,
			clusterDSNs:  []string{"host1", "host2"},
			dialTimeout:  3000 * time.Millisecond,
			currentIndex: 0,
		}

		mockDriver, err := connector.Connect(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, mockDriver)
	})

	t.Run("When all connector failed to connect, then return error", func(t *testing.T) {
		mockConnectors := []driver.Connector{
			&MockConnector{shouldFail: true},
			&MockConnector{shouldFail: true},
		}

		connector := &mysqlMultiHostConnector{
			mu:           &sync.Mutex{},
			connectors:   mockConnectors,
			clusterDSNs:  []string{"host1", "host2"},
			dialTimeout:  3000 * time.Millisecond,
			currentIndex: 0,
		}

		mockDriver, err := connector.Connect(ctx)
		assert.Error(t, err)
		assert.Nil(t, mockDriver)
	})
}
