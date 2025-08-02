package storage

// SQLCreateEventsTable defines the schema and index for the kube_events table.
const SQLCreateEventsTable = `
CREATE TABLE IF NOT EXISTS kube_events (
	-- From metav1.TypeMeta (inlined)
	kind VARCHAR,
	apiVersion VARCHAR,

	-- From metav1.ObjectMeta
	metadata STRUCT(name VARCHAR, namespace VARCHAR, uid VARCHAR, resourceVersion VARCHAR, creationTimestamp TIMESTAMP),

	-- From corev1.Event
	involvedObject STRUCT(kind VARCHAR, namespace VARCHAR, name VARCHAR, uid VARCHAR, apiVersion VARCHAR, resourceVersion VARCHAR, fieldPath VARCHAR),
	reason VARCHAR, message VARCHAR,
	source STRUCT(component VARCHAR, host VARCHAR),
	firstTimestamp TIMESTAMP, lastTimestamp TIMESTAMP,
	"count" INTEGER, "type" VARCHAR, eventTime TIMESTAMP,
	series STRUCT("count" INTEGER, lastObservedTime TIMESTAMP) DEFAULT NULL,
	action VARCHAR,
	related STRUCT(kind VARCHAR, namespace VARCHAR, name VARCHAR, uid VARCHAR, apiVersion VARCHAR, resourceVersion VARCHAR, fieldPath VARCHAR) DEFAULT NULL,
	reportingComponent VARCHAR, reportingInstance VARCHAR
);
CREATE UNIQUE INDEX IF NOT EXISTS kube_events_resourceVersion_idx ON kube_events (((metadata).resourceVersion));
`

// SQLCountKubeEvents counts the total number of rows in the kube_events table.
const SQLCountKubeEvents = `SELECT COUNT(*) FROM kube_events`

// SQLDropResourceVersionIndex drops the unique index on the resourceVersion field.
const SQLDropResourceVersionIndex = `DROP INDEX IF EXISTS kube_events_resourceVersion_idx`

// SQLInsertEventValuesTemplate is the template for the VALUES placeholder for inserting data into the kube_events table.
const SQLInsertEventValuesTemplate = `INSERT OR IGNORE INTO kube_events VALUES %s`

// SQLRenameTableToTemplate is the template for renaming the kube_events table.
const SQLRenameTableToTemplate = `ALTER TABLE kube_events RENAME TO %s`

// SQLSelectMinMaxTimestampTemplate is the template for selecting min/max timestamps from a table.
const SQLSelectMinMaxTimestampTemplate = `SELECT MIN(lastTimestamp), MAX(lastTimestamp) FROM %s`

// SQLCopyToParquetTemplate is the template for copying a table to a compressed Parquet file.
const SQLCopyToParquetTemplate = `COPY %s TO '%s' (FORMAT 'parquet', COMPRESSION 'zstd')`

// SQLDropTableTemplate is the template for dropping a table.
const SQLDropTableTemplate = `DROP TABLE %s`