package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/marcboeker/go-duckdb/v2"
	corev1 "k8s.io/api/core/v1"
)

func NewDB(path string) (*sql.DB, *duckdb.Connector, error) {
	connector, err := duckdb.NewConnector(path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create duckdb connector: %w", err)
	}

	db := sql.OpenDB(connector)
	return db, connector, nil
}

func CreateTable(db *sql.DB) error {
	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS kube_events (
		-- From metav1.TypeMeta (inlined)
		kind VARCHAR,
		apiVersion VARCHAR,
	
		-- From metav1.ObjectMeta
		metadata STRUCT(
			name VARCHAR,
			namespace VARCHAR,
			uid VARCHAR,
			resourceVersion VARCHAR,
			creationTimestamp TIMESTAMP
		),
	
		-- From corev1.Event
		involvedObject STRUCT(
			kind VARCHAR,
			namespace VARCHAR,
			name VARCHAR,
			uid VARCHAR,
			apiVersion VARCHAR,
			resourceVersion VARCHAR,
			fieldPath VARCHAR
		),
		reason VARCHAR,
		message VARCHAR,
		source STRUCT(
			component VARCHAR,
			host VARCHAR
		),
		firstTimestamp TIMESTAMP,
		lastTimestamp TIMESTAMP,
		"count" INTEGER,
		"type" VARCHAR,
		eventTime TIMESTAMP,
		series STRUCT(
			"count" INTEGER,
			lastObservedTime TIMESTAMP
		) DEFAULT NULL,
		action VARCHAR,
		related STRUCT(
			kind VARCHAR,
			namespace VARCHAR,
			name VARCHAR,
			uid VARCHAR,
			apiVersion VARCHAR,
			resourceVersion VARCHAR,
			fieldPath VARCHAR
		) DEFAULT NULL,
		reportingComponent VARCHAR,
		reportingInstance VARCHAR
	);
	`); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

func NewAppender(ctx context.Context, connector *duckdb.Connector) (*duckdb.Appender, func(), error) {
	conn, err := connector.Connect(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection: %w", err)
	}

	appender, err := duckdb.NewAppenderFromConn(conn, "", "kube_events")
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to create appender: %w", err)
	}

	cleanup := func() {
		appender.Close()
		conn.Close()
	}

	return appender, cleanup, nil
}

func AppendEvent(appender *duckdb.Appender, k8sEvent *corev1.Event) error {
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

	err := appender.AppendRow(
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
	if err := appender.Flush(); err != nil {
		log.Printf("failed to flush appender: %v", err)
	}
	return nil
}
