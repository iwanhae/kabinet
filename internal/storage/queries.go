package storage

const createTableSQL = `
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
CREATE UNIQUE INDEX IF NOT EXISTS kube_events_resourceVersion_idx
  ON kube_events (
    ((metadata).resourceVersion)
  );
`
