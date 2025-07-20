package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"

	"github.com/marcboeker/go-duckdb/v2"
	corev1 "k8s.io/api/core/v1"
)

type Storage struct {
	db       *sql.DB
	conn     driver.Conn
	appender *duckdb.Appender
}

func New(ctx context.Context, path string) (*Storage, error) {
	connector, err := duckdb.NewConnector(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create duckdb connector: %w", err)
	}

	db := sql.OpenDB(connector)
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	conn, err := connector.Connect(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	appender, err := duckdb.NewAppenderFromConn(conn, "", "kube_events")
	if err != nil {
		conn.Close()
		db.Close()
		return nil, fmt.Errorf("failed to create appender: %w", err)
	}

	return &Storage{
		db:       db,
		conn:     conn,
		appender: appender,
	}, nil
}

func (s *Storage) Close() {
	s.appender.Close()
	s.conn.Close()
	s.db.Close()
}

func (s *Storage) AppendEvent(k8sEvent *corev1.Event) error {
	var series map[string]any
	if k8sEvent.Series != nil {
		series = map[string]any{
			"count":            k8sEvent.Series.Count,
			"lastObservedTime": k8sEvent.Series.LastObservedTime.Time,
		}
	} else {
		series = map[string]any{"count": nil, "lastObservedTime": nil}
	}

	var related map[string]any
	if k8sEvent.Related != nil {
		related = map[string]any{
			"kind":            k8sEvent.Related.Kind,
			"namespace":       k8sEvent.Related.Namespace,
			"name":            k8sEvent.Related.Name,
			"uid":             string(k8sEvent.Related.UID),
			"apiVersion":      k8sEvent.Related.APIVersion,
			"resourceVersion": k8sEvent.Related.ResourceVersion,
			"fieldPath":       k8sEvent.Related.FieldPath,
		}
	} else {
		related = map[string]any{
			"kind":            nil,
			"namespace":       nil,
			"name":            nil,
			"uid":             nil,
			"apiVersion":      nil,
			"resourceVersion": nil,
			"fieldPath":       nil,
		}
	}

	err := s.appender.AppendRow(
		k8sEvent.Kind,
		k8sEvent.APIVersion,
		map[string]any{
			"name":              k8sEvent.ObjectMeta.Name,
			"namespace":         k8sEvent.ObjectMeta.Namespace,
			"uid":               string(k8sEvent.ObjectMeta.UID),
			"resourceVersion":   k8sEvent.ObjectMeta.ResourceVersion,
			"creationTimestamp": k8sEvent.ObjectMeta.CreationTimestamp.Time,
		},
		map[string]any{
			"kind":            k8sEvent.InvolvedObject.Kind,
			"namespace":       k8sEvent.InvolvedObject.Namespace,
			"name":            k8sEvent.InvolvedObject.Name,
			"uid":             string(k8sEvent.InvolvedObject.UID),
			"apiVersion":      k8sEvent.InvolvedObject.APIVersion,
			"resourceVersion": k8sEvent.InvolvedObject.ResourceVersion,
			"fieldPath":       k8sEvent.InvolvedObject.FieldPath,
		},
		k8sEvent.Reason,
		k8sEvent.Message,
		map[string]any{
			"component": k8sEvent.Source.Component,
			"host":      k8sEvent.Source.Host,
		},
		k8sEvent.FirstTimestamp.Time,
		k8sEvent.LastTimestamp.Time,
		k8sEvent.Count,
		k8sEvent.Type,
		k8sEvent.EventTime.Time,
		series,
		k8sEvent.Action,
		related,
		k8sEvent.ReportingController,
		k8sEvent.ReportingInstance,
	)

	if err != nil {
		return fmt.Errorf("failed to append row: %w", err)
	}
	if err := s.appender.Flush(); err != nil {
		log.Printf("failed to flush appender: %v", err)
	}
	return nil
}

func (s *Storage) GetLastResourceVersion() (string, error) {
	var resourceVersion string
	// Order by eventTime DESC and then resourceVersion DESC to handle events with the same timestamp.
	// We are casting resourceVersion to a UINTEGER for sorting because Kubernetes resourceVersions are
	// large numbers represented as strings.
	err := s.db.QueryRow("SELECT metadata.resourceVersion FROM kube_events ORDER BY eventTime DESC, TRY_CAST(metadata.resourceVersion AS UINTEGER) DESC LIMIT 1").Scan(&resourceVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			// If the table is empty, we don't have a resource version to start from.
			return "", nil
		}
		return "", fmt.Errorf("failed to query last resource version: %w", err)
	}
	return resourceVersion, nil
}
