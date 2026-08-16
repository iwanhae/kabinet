import type { EventResult } from "../../../types/events";
import type { FilterField } from "../../../lib/filters/model";

export interface DetailRow {
  label: string;
  get: (e: EventResult) => string | number | undefined | null;
  /** When set, the value is clickable and adds an equality filter chip. */
  field?: FilterField;
  mono?: boolean;
}

export interface DetailSection {
  title: string;
  rows: DetailRow[];
  /** Hide the section if every row is empty. */
  optional?: boolean;
}

export const DETAIL_SECTIONS: DetailSection[] = [
  {
    title: "Summary",
    rows: [
      { label: "Type", get: (e) => e.type, field: "type" },
      { label: "Reason", get: (e) => e.reason, field: "reason", mono: true },
      { label: "Count", get: (e) => e.count ?? 1, mono: true },
      { label: "Action", get: (e) => e.action, mono: true },
    ],
  },
  {
    title: "Timestamps",
    rows: [
      { label: "First seen", get: (e) => e.firstTimestamp, mono: true },
      { label: "Last seen", get: (e) => e.lastTimestamp, mono: true },
      { label: "Event time", get: (e) => e.eventTime, mono: true },
      {
        label: "Created",
        get: (e) => e.metadata.creationTimestamp,
        mono: true,
      },
    ],
  },
  {
    title: "Involved object",
    rows: [
      { label: "Kind", get: (e) => e.involvedObject?.kind, field: "kind" },
      {
        label: "Name",
        get: (e) => e.involvedObject?.name,
        field: "objectName",
        mono: true,
      },
      {
        label: "Namespace",
        get: (e) => e.involvedObject?.namespace,
        field: "namespace",
        mono: true,
      },
      { label: "UID", get: (e) => e.involvedObject?.uid, mono: true },
      {
        label: "API version",
        get: (e) => e.involvedObject?.apiVersion,
        mono: true,
      },
      {
        label: "Field path",
        get: (e) => e.involvedObject?.fieldPath,
        mono: true,
      },
    ],
  },
  {
    title: "Metadata",
    rows: [
      { label: "Name", get: (e) => e.metadata.name, mono: true },
      {
        label: "Namespace",
        get: (e) => e.metadata.namespace,
        field: "namespace",
        mono: true,
      },
      { label: "UID", get: (e) => e.metadata.uid, mono: true },
      {
        label: "Resource version",
        get: (e) => e.metadata.resourceVersion,
        mono: true,
      },
    ],
  },
  {
    title: "Source",
    optional: true,
    rows: [
      {
        label: "Component",
        get: (e) => e.source?.component,
        field: "component",
        mono: true,
      },
      { label: "Host", get: (e) => e.source?.host, field: "host", mono: true },
      {
        label: "Reporting component",
        get: (e) => e.reportingComponent,
        mono: true,
      },
      {
        label: "Reporting instance",
        get: (e) => e.reportingInstance,
        mono: true,
      },
    ],
  },
  {
    title: "Series",
    optional: true,
    rows: [
      { label: "Count", get: (e) => e.series?.count, mono: true },
      {
        label: "Last observed",
        get: (e) => e.series?.lastObservedTime,
        mono: true,
      },
    ],
  },
  {
    title: "Related object",
    optional: true,
    rows: [
      { label: "Kind", get: (e) => e.related?.kind },
      { label: "Name", get: (e) => e.related?.name, mono: true },
      { label: "Namespace", get: (e) => e.related?.namespace, mono: true },
      { label: "UID", get: (e) => e.related?.uid, mono: true },
    ],
  },
];
