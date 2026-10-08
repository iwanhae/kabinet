import React, { useMemo } from "react";
import { useSearch } from "wouter";
import { useFilters } from "../hooks/useFilters";
import { useEventsInfinite } from "../hooks/useEventsInfinite";
import { useEventsQuery } from "../hooks/useEventsQuery";
import { useUrlParams } from "../hooks/useUrlParams";
import { escapeSqlString } from "../lib/filters/compile";
import EventsVirtualTable from "../components/explore/EventsVirtualTable";
import EventDetailPanel from "../components/explore/detail/EventDetailPanel";
import TimelineHistogram from "../components/charts/TimelineHistogram";
import { Alert } from "../ui";
import { formatCount } from "../utils/format";
import type { EventResult } from "../types/events";
import styles from "./Explore.module.css";

const Explore: React.FC = () => {
  const filters = useFilters();
  const { updateParams } = useUrlParams();
  const search = useSearch();

  const {
    events,
    loadMore,
    isLoadingInitial,
    isLoadingMore,
    isReachingEnd,
    error,
  } = useEventsInfinite(filters.whereSql);

  // Selection lives in the URL: `uid` (current) or `resourceVersion` (legacy links).
  const { uidParam, rvParam, eventTs, eventRv } = useMemo(() => {
    const params = new URLSearchParams(search);
    return {
      uidParam: params.get("uid"),
      rvParam: params.get("resourceVersion"),
      eventTs: params.get("eventTs"),
      eventRv: params.get("eventRv"),
    };
  }, [search]);

  const selectedFromList = useMemo(
    () =>
      uidParam
        ? events.find(
            (e) =>
              e.metadata.uid === uidParam &&
              (!eventRv || e.metadata.resourceVersion === eventRv),
          )
        : undefined,
    [events, uidParam, eventRv],
  );

  const lookupTime = eventTs ?? selectedFromList?.timestamp;
  const lookupQuery = uidParam
    ? `SELECT * FROM $events WHERE metadata.uid = '${escapeSqlString(uidParam)}'${eventRv ? ` AND metadata.resourceVersion = '${escapeSqlString(eventRv)}'` : ""} LIMIT 1`
    : !uidParam && rvParam
      ? `SELECT * FROM $events WHERE metadata.resourceVersion = '${escapeSqlString(rvParam)}' LIMIT 1`
      : null;
  const { data: lookupData } = useEventsQuery<EventResult>(lookupQuery, {
    scope: "detail",
    ...(lookupTime ? { from: lookupTime, to: lookupTime } : {}),
  });

  // SWR keeps the previous response while a new selection is loading.
  const candidate = lookupData?.[0];
  const selected =
    candidate &&
    (uidParam
      ? candidate.metadata.uid === uidParam &&
        (!eventRv || candidate.metadata.resourceVersion === eventRv)
      : candidate.metadata.resourceVersion === rvParam)
      ? candidate
      : null;
  const panelOpen = Boolean(uidParam || rvParam);

  return (
    <div className={styles.page}>
      {error && <Alert tone="error">Query failed: {error.message}</Alert>}

      <div className={styles.timelineCard}>
        <TimelineHistogram height={120} />
      </div>

      <div className={styles.resultsMeta}>
        {isLoadingInitial
          ? "loading…"
          : `${formatCount(events.length)} events loaded${
              isReachingEnd ? " · end of range" : ""
            }`}
      </div>

      <div className={styles.tableWrap}>
        <EventsVirtualTable
          events={events}
          onRowClick={(e) =>
            updateParams({
              uid: e.metadata.uid,
              eventTs: e.timestamp,
              eventRv: e.metadata.resourceVersion,
              resourceVersion: undefined,
            })
          }
          onEndReached={loadMore}
          isLoadingMore={isLoadingMore}
          isReachingEnd={isReachingEnd}
          selectedUid={uidParam ?? undefined}
        />
      </div>

      <EventDetailPanel
        open={panelOpen}
        event={selected}
        onClose={() =>
          updateParams({
            uid: undefined,
            eventTs: undefined,
            eventRv: undefined,
            resourceVersion: undefined,
          })
        }
        onFilter={(field, value) =>
          filters.addChip({ field, op: "eq", values: [value] })
        }
      />
    </div>
  );
};

export default Explore;
