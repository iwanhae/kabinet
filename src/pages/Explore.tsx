import React, { useMemo } from "react";
import { useSearch } from "wouter";
import { useFilters } from "../hooks/useFilters";
import { useSortState } from "../hooks/useSortState";
import { useEventsInfinite } from "../hooks/useEventsInfinite";
import { useEventsQuery } from "../hooks/useEventsQuery";
import { useTimeRange, useUrlParams } from "../hooks/useUrlParams";
import { escapeSqlString } from "../lib/filters/compile";
import FilterBar from "../components/explore/FilterBar";
import EventsVirtualTable from "../components/explore/EventsVirtualTable";
import EventDetailPanel from "../components/explore/detail/EventDetailPanel";
import TimelineHistogram from "../components/charts/TimelineHistogram";
import { Alert } from "../ui";
import { formatCount } from "../utils/format";
import type { EventResult } from "../types/events";
import styles from "./Explore.module.css";

const Explore: React.FC = () => {
  const filters = useFilters();
  const { sort, toggleSort } = useSortState();
  const { from, to } = useTimeRange();
  const { updateParams } = useUrlParams();
  const search = useSearch();

  const {
    events,
    loadMore,
    isLoadingInitial,
    isLoadingMore,
    isReachingEnd,
    error,
  } = useEventsInfinite(filters.whereSql, sort);

  // Selection lives in the URL: `uid` (current) or `resourceVersion` (legacy links).
  const { uidParam, rvParam } = useMemo(() => {
    const params = new URLSearchParams(search);
    return {
      uidParam: params.get("uid"),
      rvParam: params.get("resourceVersion"),
    };
  }, [search]);

  const selectedFromList = useMemo(
    () =>
      uidParam ? events.find((e) => e.metadata.uid === uidParam) : undefined,
    [events, uidParam],
  );

  const lookupQuery =
    uidParam && !selectedFromList
      ? `SELECT * FROM $events WHERE metadata.uid = '${escapeSqlString(uidParam)}' LIMIT 1`
      : !uidParam && rvParam
        ? `SELECT * FROM $events WHERE metadata.resourceVersion = '${escapeSqlString(rvParam)}' LIMIT 1`
        : null;
  const { data: lookupData } = useEventsQuery<EventResult>(lookupQuery, {
    scope: "detail",
  });

  const selected = selectedFromList ?? lookupData?.[0] ?? null;
  const panelOpen = Boolean(uidParam || rvParam);

  const downloadHref = `/download?${new URLSearchParams({
    where: filters.whereSql,
    from,
    to,
  }).toString()}`;

  return (
    <div className={styles.page}>
      <FilterBar
        state={filters.state}
        onAddChip={filters.addChip}
        onUpdateChip={filters.updateChip}
        onRemoveChip={filters.removeChip}
        onSetRawMode={filters.setRawMode}
        onSetChipsMode={filters.setChipsMode}
        onClear={filters.clear}
        downloadHref={downloadHref}
      />

      {error && <Alert tone="error">Query failed: {error.message}</Alert>}

      <div className={styles.timelineCard}>
        <TimelineHistogram where={filters.whereSql} height={120} />
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
          sort={sort}
          onSortChange={toggleSort}
          onRowClick={(e) =>
            updateParams({ uid: e.metadata.uid, resourceVersion: undefined })
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
          updateParams({ uid: undefined, resourceVersion: undefined })
        }
        onFilter={(field, value) =>
          filters.addChip({ field, op: "eq", values: [value] })
        }
      />
    </div>
  );
};

export default Explore;
